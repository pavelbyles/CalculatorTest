package hello

import (
	"context"
	"encoding/base64"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/mux"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/googleapi"
	pubsub "google.golang.org/api/pubsub/v1beta2"
	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
)

var templates *template.Template

var projectName string
var topicName string

var pubsubTopicID string

//const PubsubTopicID = "projects/calculator-test-182623/topics/calcfinished"

type badRequest struct{ error }
type notFound struct{ error }

func init() {
	templates = template.Must(template.
		ParseFiles(
			"views/layouts/main.html",
			"views/home/result.html",
			"views/home/index.html"))

	if projectName = os.Getenv("PROJECT_NAME"); projectName == "" {
		return
	}

	if topicName = os.Getenv("RESULT_TOPIC"); topicName == "" {
		return
	}
	pubsubTopicID = fullTopicName(projectName, topicName)

	r := mux.NewRouter()
	r.HandleFunc("/", errorHandler(rootHandler)).Methods("GET")
	r.HandleFunc("/add", errorHandler(addHandler)).Methods("POST")
	r.HandleFunc("/getLastResult", errorHandler(getPubsubMessageItemHandler)).Methods("GET")

	http.Handle("/", r)
}

func rootHandler(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "text/html")
	m := &indexVM{Title: "Main Index Page", PageHeading: "Calculate Result"}
	return renderIndexTmpl(w, m)
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
	m := &resultVM{Result: strconv.Itoa(result),
		PageHeading: "Result is!..."}
	return renderResultTmpl(w, m)
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
			http.Error(w, "oops: "+err.Error(), http.StatusInternalServerError)
		}
	}

}

func renderIndexTmpl(w http.ResponseWriter, m *indexVM) error {
	return templates.ExecuteTemplate(w, "index", m)
}

func renderResultTmpl(w http.ResponseWriter, m *resultVM) error {
	return templates.ExecuteTemplate(w, "result", m)
}
