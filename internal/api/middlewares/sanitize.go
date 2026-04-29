package middlewares

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"

	"net/http"
	"net/url"
	"restapi/pkg/utils"

	"github.com/microcosm-cc/bluemonday"
)

func XSSMiddleware(next http.Handler) http.Handler{
	fmt.Println("****** Initializing XSS MIDDLEWARE")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request){
		fmt.Println("*(***** XSSMIDDLEWAER RAN)")
// sanitize the url path
		sanitizedPath,err := clean(r.URL.Path)
		if err!=nil{
			http.Error(w,err.Error(),http.StatusBadRequest)
			return 
		}
		fmt.Println("origina path",r.URL.Path)
		fmt.Println("origina path",sanitizedPath)


	// sanitize query params
	params := r.URL.Query()
	sanitizedQuery := make(map[string][]string)
	for key,values := range params{
		sanitizedKey,err := clean(key)
		if err!=nil{
			http.Error(w,err.Error(),http.StatusBadRequest)
			return 
		}
		var sanitizedValues []string
		for _,value := range values{
			cleanValue,err := clean(value)
			if err!=nil{
			http.Error(w,err.Error(),http.StatusBadRequest)
			return 
		}
		sanitizedValues = append(sanitizedValues, cleanValue.(string))
		}
		sanitizedQuery[sanitizedKey.(string)] = sanitizedValues
		fmt.Printf("Original query: %s : %s",key,strings.Join(values,""))
		fmt.Printf("Sanitized query: %s : %s",sanitizedKey,strings.Join(sanitizedValues,""))
	}

	r.URL.Path = sanitizedPath.(string)
	r.URL.RawQuery = url.Values(sanitizedQuery).Encode()
	fmt.Println("Updated URL:",r.URL.String())

	// Sanitize request body
	if r.Header.Get("Content-Type") == "application/json" {
		if r.Body != nil {
			bodyBytes, err := io.ReadAll(r.Body)
			if err != nil {
				utils.ErrorHandler(err, "")
				http.Error(w, "Error reading request body", http.StatusBadRequest)
				return
			}

			bodyString := strings.TrimSpace(string(bodyBytes))
			fmt.Println("Original Body:", bodyString)

			r.Body = io.NopCloser(bytes.NewReader([]byte(bodyString)))

			if len(bodyString) > 0 {
				var inputData interface{}
				err:= json.NewDecoder(bytes.NewReader([]byte(bodyString))).Decode(&inputData)
				if err != nil {
				http.Error(w, "Inalid json request body", http.StatusBadRequest)
				return
			}
			fmt.Println("Original body",inputData)
			//sanitize

			sanitizedData,err := clean(inputData)
				if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			fmt.Println("Sanitized json body", sanitizedData)

			// Marshal the sanitized data back to the body
			sanitizedBody, err := json.Marshal(sanitizedData)
			if err != nil {
				http.Error(w, utils.ErrorHandler(err, "Error sanitizing body").Error(), http.StatusBadRequest)
				return
			}

			r.Body = io.NopCloser(bytes.NewReader(sanitizedBody))
			fmt.Println("Sanitized body:", string(sanitizedBody))

			}else{
				log.Println("Request body is empty")
			}

		} else {
			log.Println("No body in the request")
		}
	} else if r.Header.Get("Content-Type")!="" {
		log.Printf("Received request with unsupported Content-Type: %s. Expected application/json.\n", r.Header.Get("Content-Type"))
		http.Error(w, "Unsupported Content-Type. Please use application/json.", http.StatusUnsupportedMediaType)
		return
	}

		next.ServeHTTP(w,r)
		fmt.Println("*(***** SENDING RESPONSE FROM XSSMIDDLEWAER RAN)")
	})
}

func clean(data interface{}) (interface{},error){

	switch v:= data.(type){
	case map[string]interface{}:
		for key,value := range v{
			v[key] = sanitizeValue(value)
		}
		return  v,nil
	case []interface{}:
		for i,value := range v{
			v[i] = sanitizeValue(value)
		}
		return  v,nil
	
	case string:
		return  sanitizeString(v),nil

	default:
		//error
		return nil,utils.ErrorHandler(fmt.Errorf("unssuported type:%T",data),fmt.Sprintf("Unssuoprted data: %T",data))
	}
}

func sanitizeValue(data interface{}) interface{}{
	switch v :=data.(type) {
	case map[string]interface{}:
		for k,value := range v{
			v[k] = sanitizeValue(value)
		}
		return v
	case string:
		return  sanitizeString(v)

	case []interface{}:
		for i,value := range v{
			v[i] = sanitizeValue(value)
		}
		return v
	default:
		return v
	}

}

func sanitizeString(value string)string{
	return bluemonday.UGCPolicy().Sanitize(value)
}