/*
AUTHOR: Mario Reiley
NOTE: it is my port to Golang 1.19

# URL Shortener Microservice

Build a full stack JavaScript app that is functionally similar to this: https://url-shortener-microservice.freecodecamp.rocks.

  - You should provide your own project, not the example URL.
  - You can POST a URL to /api/shorturl and get a JSON response with original_url and short_url properties.
    Here's an example: { original_url : 'https://freeCodeCamp.org', short_url : 1}
  - When you visit /api/shorturl/<short_url>, you will be redirected to the original URL.
  - If you pass an invalid URL that doesn't follow the valid http://www.example.com format,
    the JSON response will contain { error: 'invalid url' }
*/
package main

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"strconv"
)

const SIZE = 16

type idType int64

type urlJsonResponse struct {
	Original_url string `json:"original_url"`
	Short_url    idType `json:"short_url"`
}

type BodySent struct {
	Url string
}

type urlErrors struct {
	Msj string `json:"msg"`
}

/*
Since binary encoding do not work for string types them i had create a index for store
the real data length for later recover and launch it URL. finannly I used two diferentes
encoding systems.

index.dat use gob encoding
shorurl.dat use binary encoding

just is a little trick!
*/
type Index struct {
	Id    idType
	Chunk int64
}

type ShortUrl struct {
	Id  idType
	Url string
}

// Just get a list id for one moment
func fill(rs *map[idType]int64) {

	f, e := os.OpenFile("index.dat", os.O_RDONLY, os.ModePerm)
	if e != nil {
		return // no data!
	}

	defer func() {
		if e != nil {
			f.Close()
		}
	}()

	for done := true; done == true; {
		data := make([]byte, SIZE)
		_, e := f.Read(data)
		if e != nil || e == io.EOF {
			done = false
			continue
		}
		r := Index{}
		buf := bytes.NewBuffer(data)
		binary.Read(buf, binary.LittleEndian, &r)
		(*rs)[r.Id] = r.Chunk
	}

}

/*
for each id, we need the real length of the data so it is store in a index.dat file
because binary encode no support string data. this function use two kind encode
sistem. binary and gob.
*/
func add(id idType, body BodySent, ok chan error) {
	var dataIndex []byte
	var dataShort []byte

	// Manualy calculate the real chunk bytes
	chunk := ShortUrl{Id: id, Url: body.Url}
	bufShort := bytes.NewBuffer(dataShort)
	enco := gob.NewEncoder(bufShort)
	if e := enco.Encode(chunk); e != nil {
		ok <- fmt.Errorf("add: Manualy calculate the real chunk bytes %v:", e)
		return
	}
	dataShort = bufShort.Bytes()

	fin, e := os.OpenFile("index.dat", os.O_CREATE|os.O_APPEND|os.O_RDWR, os.ModePerm)
	if e != nil {
		ok <- fmt.Errorf("add: Open index.dat %v:", e)
		return
	}

	defer func() {
		fin.Close()
	}()

	bufIndex := bytes.NewBuffer(dataIndex)
	record := Index{Id: id, Chunk: int64(binary.Size(dataShort))}
	binary.Write(bufIndex, binary.LittleEndian, record)
	/*
		Just asegurate id and chunk. never main if write the short url to disk fail.
		the point is get unique id.

	*/
	dataIndex = bufIndex.Bytes()
	if _, e := fin.Write(dataIndex); e != nil {
	} else {
		fsh, e := os.OpenFile("shorturl.dat", os.O_CREATE|os.O_APPEND|os.O_WRONLY, os.ModePerm)
		if e != nil {
			ok <- fmt.Errorf("add: Open shorturl.dat %v:", e)
			return
		} else {
			if _, e := fsh.Write(dataShort); e != nil {
				ok <- fmt.Errorf("add: Write dataShort %v:", e)
			} else {
				fsh.Close()
				return
			}
		}
	}

	ok <- nil // all be fine

	return
}

func createId(body *BodySent) (Id idType, err error) {

	var records = make(map[idType]int64)
	// create a index of uniques Ids for one MOMEMT!
	fill(&records)
	if len(records) > 0 {
		// create an unique ids
		for {
			id := idType(math.Floor(rand.Float64() * 100))
			if _, exist_ := records[id]; exist_ == true {
				continue // search other Id
			}
			check := make(chan error)
			go add(id, *body, check)
			select {
			case err = <-check:
				if err != nil {
					return 0, err
				}
			}
			Id = id
			break
		}
	} else { // Firts id

		check := make(chan error)
		id := idType(math.Floor(rand.Float64() * 100))
		go add(id, *body, check)
		select {
		case err = <-check:
			if err != nil {
				return 0, err
			}
		}
		Id = id

	}

	return Id, nil
}

/*
Get parameters comming from resquest,Validate it and securate an unique Id
*/
var UpdateUrl = func(w http.ResponseWriter, req *http.Request) {
	var body BodySent

	dataBody, _ := io.ReadAll(req.Body)
	// Applicate cache ?

	if err := json.Unmarshal(dataBody, &body); err != nil {
		NoUnmarshal := urlErrors{Msj: "can not Unmarshal data body"}
		res, _ := json.Marshal(NoUnmarshal)
		w.Write(res)
		return
	}

	// No perfect but work
	if re := regexp.MustCompile(`^https?:\/\/`); re.MatchString(body.Url) == false {
		Invalid := urlErrors{Msj: "Invalid url"}
		res, _ := json.Marshal(Invalid)
		w.Write(res)
		return
	}

	id, err := createId(&body) // I get Id ready and secure it
	if err != nil {
		IdErr := urlErrors{Msj: err.Error()}
		res, _ := json.Marshal(IdErr)
		w.Write(res)
		return
	} else {
		dataResponse := urlJsonResponse{Original_url: body.Url, Short_url: id}
		res, _ := json.Marshal(dataResponse)
		w.Write(res)
		return

	}

}

var DistpathURL = func(w http.ResponseWriter, req *http.Request) {

	var records = make(map[idType]int64)

	fill(&records)
	param := req.URL.Query().Get("id")
	id, _ := strconv.Atoi(param)
	if _, exist_ := records[idType(id)]; exist_ == true {

		var r ShortUrl
		f, err := os.Open("shorturl.dat")
		if err != nil {
			fmt.Println("error al abrir")
		}

		defer func() {
			f.Close()
		}()

		for _, v := range records {
			data := make([]byte, v)
			if _, err := f.Read(data); err != nil || err == io.EOF {
				break
			} else {
				buf := bytes.NewBuffer(data)
				deco := gob.NewDecoder(buf)
				deco.Decode(&r)
				// launch the Url via short Id
				if r.Id == idType(id) {
					http.Redirect(w, req, r.Url, http.StatusFound)
					break
				}

			}

		}

	} else {
		invalid := urlErrors{Msj: "Short Id do not exist"}
		res, _ := json.Marshal(invalid)
		w.Write(res)
		return
	}
}

func main() {
	/*
		POST /api/shorturl/ must be receive a json via body package for example:
		{"url":"www.google.com"}

	*/
	http.HandleFunc("/api/shorturl/", UpdateUrl)
	http.HandleFunc("/api/shorturl/short/", DistpathURL)
	log.Fatal(http.ListenAndServe("localhost:8080", nil))

}
