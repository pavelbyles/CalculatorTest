package hello

import (
	"encoding/base64"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"golang.org/x/oauth2/google"
	pubsub "google.golang.org/api/pubsub/v1beta2"
	"google.golang.org/appengine"
	"google.golang.org/api/googleapi"
)

var templates *template.Template
const ProjectName = "calculator-test-182623"
const TopicName = "calcfinished"

var PubsubTopicID = fullTopicName(ProjectName, TopicName)

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
	t = template.new("root")
	t, _ = t.Parse(addTemplate)
	t.Execute(w)

	return nil
}

func addHandler(w http.ResponseWriter, r *http.Request) error {
	ctx := appengine.NewContext(r)
	s1 := r.FormValue("number1")
	s2 := r.FormValue("number2")

	if len(s1) < 1 || len(s2) < 1 {
		w.WriteHeader(500)
		return nil
	}

	n1, err := strconv.Atoi(s1)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprint(w,"Error occurred converting to number: %v", s1, err)
		return nil
	}
	n2, err := strconv.Atoi(s2)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprint(w,"Error occurred converting to number: %v", s2, err)
		return nil
	}
	result := n1 + n2

	client, err := google.DefaultClient(ctx, pubsub.PubsubScope)
	if err != nil {
		fmt.Fprint(w, "Unable to set default credentials: %v", err)
		w.WriteHeader(200)
		return nil
	}

	pubsubService, err := pubsub.New(client)

	if err != nil {
		fmt.Fprint(w, "Unable to create pubsub service: %v", err)
		w.WriteHeader(200)
		return nil
	}

	if nil != err {
		fmt.Fprint(w, "Unable to create OAuth2 service: %v", err)
		w.WriteHeader(200)
		return nil
	}
	_, err = pubsubService.Projects.Topics.Create(PubsubTopicID, &pubsub.Topic{}).Do()
	if err != nil {
		switch t := err.(type) {
		default:
			fmt.Fprint(w, "createTopic Create().Do() failed: %v, %v", err, t)
			w.WriteHeader(500)
			return nil
		case *googleapi.Error:
			serr, _ := err.(*googleapi.Error)
			if serr.Code == 409 {
				log.Println("Topic already created ... continuing: ", err)
			}
		}
	}

	pubsubMessage := &pubsub.PubsubMessage{
		Data: base64.StdEncoding.EncodeToString([]byte(strconv.Itoa(result))),
	}
	publishRequest := &pubsub.PublishRequest{
		Messages: []*pubsub.PubsubMessage{pubsubMessage},
	}
	if _, err := pubsubService.Projects.Topics.Publish(PubsubTopicID, publishRequest).Do(); err != nil {
		fmt.Fprint(w, "addResultHandler Publish().Do() failed: %v", err)
		w.WriteHeader(200)
		return nil
	}

	fmt.Fprint(w, "Published a message to the topic")

	w.WriteHeader(200)
	return nil
}

func getPubsubMessageItemHandler(w http.ResponseWriter, r *http.Request) error {
	return nil
}

func fullTopicName(proj, topic string) string {
	return fqrn("topics", proj, topic)
}

func fqrn(res, proj, name string) string {
	return fmt.Sprintf("projects/%s/%s/%s", proj, res, name)
}

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
			log.Println(err)
			http.Error(w, "oops", http.StatusInternalServerError)
		}
	}

}

const addTemplate = `
<html>
<head>
  <title>Add Page</title>
</head>
<body>
  <form method="POST" action="/add">
	<label for="number1">Number 1</label><input type="text" id="number1" /><span>+</span>
    <label for="number2">Number 1</label><input type="text" id="number2" />
	<input type="submit" />
  </form>
</body>
</html>`

const resultTemplate = `
<html>
<head>
  <title>Result Page</title>
</head>
<body>
  <h1>Result</h1>
  <div>{{ . }}
</body>
</html>`