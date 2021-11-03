package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
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
	//for i := 0; i < 20; i++ {
	os.MkdirAll("./imgs/", os.ModePerm)
	var fileName string
	c := colly.NewCollector()
	ID := genID()
	//var downloaded bool
	println(ID)
	c.OnHTML("#screenshot-image", func(e *colly.HTMLElement) {
		split := strings.Split(e.Attr("src"), "/")
		fileName = split[len(split)-1]
		err := downloadFile(e.Attr("src"), fileName)
		//downloaded = true
		if err != nil {
			log.Fatal(err)
			//downloaded = false
		}
	})
	// if downloaded {
	// 	//continue
	// }
	c.Visit("https://prnt.sc/" + ID)
	fileData, err := ioutil.ReadFile(filepath.Join("./imgs/", fileName))
	if err != nil {
		log.Println("Couldn't open the file.")
		log.Fatal(err)
		//continue

	}
	httpClient := initHTTPClient(2000, 2000)
	req, err := buildRequest("http://localhost:8080/sync", "http://localhost:8080/sync", fileData)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	response, err := httpClient.Do(req)
	if response != nil {
		defer response.Body.Close()
	}
	if err != nil {
		log.Fatal(err)
	}
	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}
	if response.StatusCode != http.StatusOK {
		log.Fatal("Request failed")
	}
	d := json.NewDecoder(strings.NewReader(string(responseBody)))
	d.UseNumber()
	var answ interface{}
	if err := d.Decode(&answ); err != nil {
		log.Fatal(err)
	}
	ans, _ := answ.(map[string]interface{})
	an, _ := ans["prediction"].(map[string]interface{})
	a, _ := an["file_name"].(map[string]interface{})
	println(reflect.TypeOf(a["safe"]))
	//num, _ := a["safe"].(float64)
	// println(num)
	// if num > 0.2 {
	// 	fmt.Println("Found nudity, saving.")
	// } else {
	// 	fmt.Println("Not nudity, deleting.")
	// 	os.Remove(filepath.Join("./imgs/", fileName))
	// }

	//}
}
