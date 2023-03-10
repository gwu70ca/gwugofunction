package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"

	gofunc "github.com/gwu70ca/azuregofunctionhelper"
)

const (
	//Binding names found in function.json
	eventHubBindingName = "gwuEventHubMessages"
	queueBindingName    = "gwuQueueItem"
	blobBindingName     = "gwuBlob"
)

func homeHandler(w http.ResponseWriter, r *http.Request) {
	_log(r, "Home")

	if r.Method == "GET" {
		if tmpl, err := template.New("greeting").Parse(greetingPage); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
		} else {
			tmpl.Execute(w, nil)
		}
		return
	}

	http.Error(w, "Method not supported.", http.StatusMethodNotAllowed)
}

func timerTriggerHandler(w http.ResponseWriter, r *http.Request) {
	_log(r, "Timer trigger started")

	client := &http.Client{}
	req, _ := http.NewRequest("GET", "https://azurecomcdn.azureedge.net/en-us/blog/feed/", nil)
	req.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/110.0.0.0 Safari/537.36")
	resp, err := client.Do(req)

	fmt.Println(resp.StatusCode)
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}
	defer resp.Body.Close()

	xmlString := string(b)

	source := Rss{}
	if err := xml.Unmarshal([]byte(xmlString), &source); err != nil {
		fmt.Println(err.Error())
	}

	itemCount := 0
	for _, i := range source.Channel.Items {
		fmt.Println("-----")
		fmt.Println(i.Title)
		fmt.Println(i.Link)
		fmt.Println(i.PubDate)
		itemCount = itemCount + 1
	}

	logs := []string{
		fmt.Sprintf("Last build date: %v", source.Channel.LastBuildDate),
		fmt.Sprintf("%d item(s) read", itemCount),
	}
	invokeResponse := gofunc.InvokeResponse{
		//Outputs:     outputs,
		Logs:        logs,
		ReturnValue: "Timer triggered"}

	js, err := json.Marshal(invokeResponse)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

//Triggered by new message in storage queue and sends to 1 another storage queue
func queueTriggerHandler(w http.ResponseWriter, r *http.Request) {
	_log(r, "queueTriggerHandler")

	invokeReq, err := gofunc.ParseFunctionHostRequest(w, r)
	if err != nil {
		// bad JSON or unrecognized json field
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	//Get message from incoming request
	msg := gofunc.QueueMessage(invokeReq, queueBindingName)
	//Generate output message
	outputMsg := fmt.Sprintf("Processed message: [%v]", msg)
	//Create log
	logs := []string{outputMsg, "Binding: " + queueBindingName}
	//Generate InvokeResponse
	invokeResponse := gofunc.InvokeResponse{
		Logs:        logs,
		ReturnValue: outputMsg}

	//Send InvokeResponse back to function host
	js, err := json.Marshal(invokeResponse)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

//Triggered by new message in storage queue and sends to 3 others storage queues
func queueTriggerWithOutputsHandler(w http.ResponseWriter, r *http.Request) {
	_log(r, "queueTriggerWithOutputsHandler")

	invokeReq, err := gofunc.ParseFunctionHostRequest(w, r)
	if err != nil {
		// bad JSON or unrecognized json field
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	//Get message from incoming request
	msg := gofunc.QueueMessage(invokeReq, queueBindingName)
	//Generate output message, this will be sent to original/1st output queue defined in function.json.
	outputMsg := fmt.Sprintf("Processed message: [%v]", msg)
	returnValue := outputMsg

	//This goes to the 2nd and 3rd output queue defined in function.json
	outputs := make(map[string]interface{})
	outputs["output1"] = fmt.Sprintf("Output1: [%v]", outputMsg)
	outputs["output2"] = fmt.Sprintf("Output2: [%v]", outputMsg)

	invokeResponse := gofunc.InvokeResponse{
		Outputs:     outputs,
		Logs:        []string{outputMsg, "Binding: " + queueBindingName},
		ReturnValue: returnValue}

	js, err := json.Marshal(invokeResponse)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func blobTriggerHandler(w http.ResponseWriter, r *http.Request) {
	_log(r, "blobTriggerHandler")

	invokeReq, err := gofunc.ParseFunctionHostRequest(w, r)
	if err != nil {
		// bad JSON or unrecognized json field
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	returnValue := gofunc.BlobData(invokeReq, blobBindingName)

	blobName := gofunc.BlobName(invokeReq)
	blobUri := gofunc.BlobUri(invokeReq)
	// This appears in the log
	logs := []string{
		"Name: " + blobName,
		"Uri :" + blobUri,
	}
	invokeResponse := gofunc.InvokeResponse{Logs: logs, ReturnValue: returnValue}

	js, err := json.Marshal(invokeResponse)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func eventHubTriggerHandler(w http.ResponseWriter, r *http.Request) {
	_log(r, "eventHubTriggerHandler")

	invokeReq, err := gofunc.ParseFunctionHostRequest(w, r)
	if err != nil {
		// bad JSON or unrecognized json field
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	//Found in function.json
	inMsg := gofunc.EventHubMessage(invokeReq, eventHubBindingName)
	outMsg := "Processed [" + inMsg + "]"
	logs := []string{fmt.Sprintf("Event hub message [%v]", inMsg)}
	invokeResponse := gofunc.InvokeResponse{Logs: logs, ReturnValue: outMsg}

	js, err := json.Marshal(invokeResponse)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func httpTriggerWithOutputs(w http.ResponseWriter, r *http.Request) {
	t := _log(r, "HttpTriggerWithOutputs")

	invokeReq, err := gofunc.ParseFunctionHostRequest(w, r)
	if err != nil {
		// bad JSON or unrecognized json field
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	fmt.Println(invokeReq)
	/*
		dataHttpRequest := gofunc.HttpRequestData(invokeReq)
		fmt.Println("URL:", dataHttpRequest.URL)
		fmt.Println("METHOD:", dataHttpRequest.Method)
		fmt.Println("HEADERS:", dataHttpRequest.Header)
		fmt.Println("URL.QUERY:", dataHttpRequest.URL.Query())
	*/

	movieStar := make(map[string]interface{})
	movieStar["name"] = "Tom Hanks"
	movieStar["movie"] = map[string]interface{}{
		"Charlie Wilson's War": "Charlie Wilson",
		"Forrest Gump":         "Forrest Gump",
		"Saving Private Ryan":  "John Miller",
		"Toy Story":            "Woody",
	}

	headers := make(map[string]interface{})
	headers["server"] = "AzureFunction"
	headers["age"] = "123456"

	//Http output
	res := make(map[string]interface{})
	res["statusCode"] = "201"
	res["body"] = movieStar
	res["headers"] = headers

	outputs := make(map[string]interface{})
	outputs["res"] = res

	//This goes to $return in function.json
	returnValue := gofunc.ReturnValue{Data: textResponse("Return val from httpTriggerWithOutputs", t)}

	invokeResponse := gofunc.InvokeResponse{
		Outputs:     outputs,
		Logs:        []string{fmt.Sprintf("URL: %v", r.URL.String()), fmt.Sprintf("METHOD: %v", r.Method)},
		ReturnValue: returnValue,
	}

	js, err := json.Marshal(invokeResponse)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func httpTriggerHandlerStringReturnValue(w http.ResponseWriter, r *http.Request) {
	t := _log(r, "httpTriggerHandlerStringReturnValue")

	invokeReq, err := gofunc.ParseFunctionHostRequest(w, r)
	if err != nil {
		// bad JSON or unrecognized json field
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	fmt.Println(invokeReq)
	/*
		dataHttpRequest := gofunc.HttpRequestData(invokeReq)
		fmt.Println("URL:", dataHttpRequest.URL)
		fmt.Println("METHOD:", dataHttpRequest.Method)
		fmt.Println("HEADERS:", dataHttpRequest.Header)
		fmt.Println("URL.QUERY:", dataHttpRequest.URL.Query())
	*/

	//Shows up as response header
	headers := make(map[string]interface{})
	headers["server"] = "AzureFunction"
	headers["age"] = "123456"

	//Http output
	res := make(map[string]interface{})
	res["statusCode"] = "201"
	res["body"] = textResponse("httpTriggerHandlerStringReturnValue", t)
	res["headers"] = headers

	outputs := make(map[string]interface{})
	outputs["res"] = res

	//queryString := queryParamsToString(dataHttpRequest)

	var buffer bytes.Buffer
	buffer.WriteString("Query string:")
	for k, v := range r.URL.Query() {
		fmt.Println("k:", k, "v:", v)
		buffer.WriteString(fmt.Sprintf("%v=%v,", k, v))
	}
	//This goes to $return in function.json
	returnValue := buffer.String()

	logs := []string{
		//fmt.Sprintf("URL: %v", dataHttpRequest.URL),
		//fmt.Sprintf("METHOD: %v", dataHttpRequest.Method),
		fmt.Sprintf("URL: %v", r.URL),
		fmt.Sprintf("METHOD: %v", r.Method),
		fmt.Sprintf("QUERY: %v", returnValue),
	}

	invokeResponse := gofunc.InvokeResponseStringReturnValue{
		Outputs:     outputs,
		Logs:        logs,
		ReturnValue: returnValue}

	js, err := json.Marshal(invokeResponse)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func simpleHttpTriggerHandler(w http.ResponseWriter, r *http.Request) {
	t := _log(r, "SimpleHttpTriggerHandler")

	queryParams := r.URL.Query()

	for k, v := range queryParams {
		fmt.Println("k:", k, "v:", v)
	}

	w.Write([]byte(textResponse("Hello World from go worker (simpleHttpTriggerHandler).", t)))
}

func textResponse(resString, timeString string) string {
	return fmt.Sprintf("%v [%v]", resString, timeString)
}

func main() {
	customHandlerPort, exists := os.LookupEnv("FUNCTIONS_CUSTOMHANDLER_PORT")
	if exists {
		fmt.Println("FUNCTIONS_CUSTOMHANDLER_PORT: " + customHandlerPort)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/api/Home", homeHandler)
	mux.HandleFunc("/TimerTrigger", timerTriggerHandler)

	mux.HandleFunc("/BlobTrigger", blobTriggerHandler)
	mux.HandleFunc("/QueueTrigger", queueTriggerHandler)
	mux.HandleFunc("/QueueTriggerWithOutputs", queueTriggerWithOutputsHandler)
	mux.HandleFunc("/EventHubTrigger", eventHubTriggerHandler)

	mux.HandleFunc("/HttpTriggerStringReturnValue", httpTriggerHandlerStringReturnValue)
	mux.HandleFunc("/HttpTriggerWithOutputs", httpTriggerWithOutputs)

	mux.HandleFunc("/api/SimpleHttpTrigger", simpleHttpTriggerHandler)
	mux.HandleFunc("/api/SimpleHttpTriggerWithReturn", simpleHttpTriggerHandler)

	fmt.Println("Go server Listening...on FUNCTIONS_CUSTOMHANDLER_PORT:", customHandlerPort)
	if err := http.ListenAndServe(":"+customHandlerPort, mux); err != nil {
		log.Fatal(err)
	}
}
