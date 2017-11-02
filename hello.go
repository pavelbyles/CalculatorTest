package hello

import (
	"context"
	"encoding/base64"
	"fmt"
	"html/template"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/googleapi"
	pubsub "google.golang.org/api/pubsub/v1beta2"
	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
)

var indexTemplate = template.Must(template.New("index").Parse(indexTemplateStr))
var resultTemplate = template.Must(template.New("result").Parse(resultTemplateStr))

const projectName = "calculator-test-182623"
const topicName = "calcfinished"

var pubsubTopicID = fullTopicName(projectName, topicName)

//const PubsubTopicID = "projects/calculator-test-182623/topics/calcfinished"

type badRequest struct{ error }
type notFound struct{ error }

func init() {
	r := mux.NewRouter()
	r.HandleFunc("/", errorHandler(rootHandler)).Methods("GET")
	r.HandleFunc("/add", errorHandler(addHandler)).Methods("POST")
	r.HandleFunc("/getLastResult", errorHandler(getPubsubMessageItemHandler)).Methods("GET")

	http.Handle("/", r)
}

func rootHandler(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "text/html")
	return indexTemplate.Execute(w, "")
}

func addHandler(w http.ResponseWriter, r *http.Request) error {
	ctx := appengine.NewContext(r)
	s1 := r.FormValue("number1")
	s2 := r.FormValue("number2")

	// set default content type
	w.Header().Set("Content-Type", "text/html")

	if len(s1) < 1 || len(s2) < 1 {
		err := fmt.Sprintf("Number 1 or number 2 is blank: number1 - %v, number 2 - %v - %v", s1, s2, r.Form["number1"])
		http.Error(w, err, http.StatusInternalServerError)
		return nil
	}

	n1, err := strconv.Atoi(s1)
	if err != nil {
		w.WriteHeader(500)
		errStr := fmt.Sprintf("Error occurred converting to number: %v", s1)
		http.Error(w, errStr, http.StatusInternalServerError)
		return nil
	}
	n2, err := strconv.Atoi(s2)
	if err != nil {
		w.WriteHeader(500)
		errStr := fmt.Sprintf("Error occurred converting to number: %v", s2)
		http.Error(w, errStr, http.StatusInternalServerError)
		return nil
	}
	result := n1 + n2

	if err = addToPubsub(ctx, pubsubTopicID, strconv.Itoa(result)); err != nil {
		w.WriteHeader(500)
		errStr := fmt.Sprintf("Error occurred converting to number: %v", s2)
		http.Error(w, errStr, http.StatusInternalServerError)
		return nil
	}

	w.WriteHeader(200)
	return resultTemplate.Execute(w, "Published a message to the topic")
}

func addToPubsub(ctx context.Context, pubsubTopicID string, message string) error {
	client, err := google.DefaultClient(ctx, pubsub.PubsubScope)
	if err != nil {
		return err
	}
	pubsubService, err := pubsub.New(client)
	if err != nil {
		return err
	}
	_, err = pubsubService.Projects.Topics.Create(pubsubTopicID, &pubsub.Topic{}).Do()
	if err != nil {
		switch t := err.(type) {
		default:
			log.Errorf(ctx, "createTopic Create().Do() failed: %v, %v", err, t)
			return nil
		case *googleapi.Error:
			serr, _ := err.(*googleapi.Error)
			if serr.Code == 409 {
				log.Infof(ctx, "Topic already created ... continuing")
			}
		}
	}

	pubsubMessage := &pubsub.PubsubMessage{
		Data: base64.StdEncoding.EncodeToString([]byte(message)),
	}
	publishRequest := &pubsub.PublishRequest{
		Messages: []*pubsub.PubsubMessage{pubsubMessage},
	}
	if _, err := pubsubService.Projects.Topics.Publish(pubsubTopicID, publishRequest).Do(); err != nil {
		log.Errorf(ctx, "addResultHandler Publish().Do() failed: %v", err)
		return nil
	}
	return nil
}

func getPubsubMessageItemHandler(w http.ResponseWriter, r *http.Request) error {

	return nil
}

func fullTopicName(proj, topic string) string {
	return fqrn("topics", proj, topic)
}

// Get fully qualified topic name
func fqrn(res, proj, name string) string {
	return fmt.Sprintf("projects/%s/%s/%s", proj, res, name)
}

// General error handler for requests
func errorHandler(f func(w http.ResponseWriter, r *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := f(w, r)
		if err == nil {
			return
		}
		switch err.(type) {
		case badRequest:
			http.Error(w, err.Error(), http.StatusBadRequest)
		case notFound:
			http.Error(w, "task not found", http.StatusNotFound)
		default:
			http.Error(w, "oops", http.StatusInternalServerError)
		}
	}

}

const indexTemplateStr = `<!doctype html>
<html>
<head>
  <title>Add Page</title>
</head>
<body>
  <form id="mainForm" method="post" action="add" accept-charset="utf-8" enctype="multipart/form-data">
	<label for="number1">Number 1:</label><input type="text" name="number1" value="1"/><span>&nbsp;+&nbsp;</span>
    <label for="number2">Number 2:</label><input type="text" name="number2" value="2"/>
	<input type="submit" value="Submit"/>
  </form>
</body>
</html>`

const resultTemplateStr = `<!doctype html>
<html>
<head>
  <title>Result Page</title>
</head>
<body>
  <h1>Result</h1>
  <div>{{ . }}
</body>
</html>`
