# URL Shortener Microservice

### www.freecodecamp.com proposes this project in JavaScript so I did it in Golang.

Description: When I did this services I could use sql/no sql or text files but I make my desition to use binary files with diferente encode
system. 

For practique diferentes Golang Packages. I used:

```
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
```
I did some concurrency too!
### Task
* You can POST a URL to /api/shorturl and get a JSON response with original_url and short_url properties. Here's an example: { original_url : 'https://freeCodeCamp.org', short_url : 1}
* When you visit /api/shorturl/<short_url>, you will be redirected to the original URL.
* If you pass an invalid URL that doesn't follow the valid http://www.example.com format, the JSON response will contain { error: 'invalid url' }

## REST API Response Format

* POST http://localhost:8080/api/shorturl/

{
  "original_url": "http://google.com",
  "short_url": 60
}

* GET http://localhost:8080/api/shorturl/short/?Id=60

