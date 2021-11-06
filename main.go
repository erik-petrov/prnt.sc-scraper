package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gocolly/colly"
)

type payload struct {
	Data    data   `json:"data"`
	Webhook string `json:"webhook"`
}

type data struct {
	Filename string `json:"file_name"`
}

func buildRequest(url, webhook string, fileData []byte) (*http.Request, error) {
	encodedData := &bytes.Buffer{}
	encoder := base64.NewEncoder(base64.StdEncoding, encodedData)
	defer encoder.Close()
	encoder.Write(fileData)
	p := payload{
		Data: data{
			Filename: encodedData.String(),
		},
		Webhook: webhook,
	}
	payloadBytes, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	return http.NewRequest(http.MethodPost, url, bytes.NewBuffer(payloadBytes))
}

func initHTTPClient(timeout, dialTimeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: time.Millisecond * timeout,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return net.DialTimeout(network, addr, time.Duration(dialTimeout*time.Millisecond))
			},
		},
	}
}

func genID() string {
	const letterAndNumsBytes = "abcdefghijklmnopqrstuvwxyz1234567890"
	b := make([]byte, 6)
	for i := range b {
		if i == 0 {
			letters := "abcdefghijklmnopqrstuvwxyz"
			b[i] = letters[rand.Intn(len(letters))]
		} else {
			b[i] = letterAndNumsBytes[rand.Intn(len(letterAndNumsBytes))]
		}
	}
	return string(b)
}

func downloadFile(URL, fileName string) error {
	//Get the response bytes from the url
	client := &http.Client{}
	req, err := http.NewRequest("GET", URL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Golang_Spider_Bot/3.0")
	response, err := client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return errors.New("received non 200 response code")
	}
	//Create a empty file
	file, err := os.Create(filepath.Join("./imgs/", fileName))
	if err != nil {
		return err
	}
	defer file.Close()

	//Write the bytes to the file
	_, err = io.Copy(file, response.Body)
	if err != nil {
		return err
	}
	return nil
}

func main() {
	for i := 0; i < 400; i++ {
		//makes a folder for the images in the files directory
		os.MkdirAll("./imgs/", os.ModePerm)

		var fileName string
		var downloaded bool

		//generates the random id and creates the scraper
		c := colly.NewCollector()
		ID := genID()
		println(ID)

		//if the scraper find an element with the id given, this function starts
		c.OnHTML("#screenshot-image", func(e *colly.HTMLElement) {
			if !strings.Contains(e.Attr("src"), "imgur") {
				split := strings.Split(e.Attr("src"), ".")
				fileName = ID + "." + split[len(split)-1]
				err := downloadFile(e.Attr("src"), fileName)
				downloaded = true
				if err != nil {
					downloaded = false
					fmt.Println(err)
				}
			} else {
				log.Println("Image not available.")
			}
		})
		c.Visit("https://prnt.sc/" + ID)
		if !downloaded {
			continue
		}

		//opens the downloaded image
		fileData, err := ioutil.ReadFile(filepath.Join("./imgs/", fileName))
		if err != nil {
			log.Println("Couldn't open the file.")
			log.Fatal(err)
			continue

		}

		//creates the client for request
		httpClient := initHTTPClient(2000, 2000)
		req, err := buildRequest("http://localhost:8080/sync", "http://localhost:8080/sync", fileData)
		if err != nil {
			log.Fatal(err)
		}

		//makes the request to the server containing the image
		req.Header.Set("Content-Type", "application/json")
		response, err := httpClient.Do(req)
		if response != nil {
			defer response.Body.Close()
		}
		if err != nil {
			log.Fatal(err)
		}

		//gets the request and if its ok, proceeds
		responseBody, err := ioutil.ReadAll(response.Body)
		if err != nil {
			log.Fatal(err)
		}
		if response.StatusCode != http.StatusOK {
			log.Fatal("Request failed")
		}

		//parsing the request, so we can check if the image is safe or not
		d := json.NewDecoder(strings.NewReader(string(responseBody)))
		d.UseNumber()
		var answ interface{}
		if err := d.Decode(&answ); err != nil {
			log.Fatal(err)
		}
		ans, _ := answ.(map[string]interface{})
		an, _ := ans["prediction"].(map[string]interface{})
		a, _ := an["file_name"].(map[string]interface{})
		f, _ := a["safe"].(json.Number)
		percentage, _ := f.Float64()
		fmt.Println(percentage)
		if percentage < 0.4 && percentage != 0 {
			fmt.Println("Found nudity, saving.")
		} else {
			fmt.Println("Not nudity, deleting.")
			os.Remove(filepath.Join("./imgs/", fileName))
		}
	}
}
