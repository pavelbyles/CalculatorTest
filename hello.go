package hello

import (
	"encoding/base64"
	"fmt"
	"github.com/gorilla/mux"
	pubsub "google.golang.org/api/pubsub/v1beta2"
	"log"
	"net/http"
)

const ProjectName = "calculator-test-182623"
const TopicName = "calcfinished"

var PubsubTopicID = fullTopicName(ProjectName, TopicName)

//const PubsubTopicID = "projects/calculator-test-182623/topics/calcfinished"

type badRequest struct{ error }
type notFound struct{ error }

func init() {
	r := mux.NewRouter()
	r.HandleFunc("/addResult", errorHandler(addResultHandler)).Methods("POST")

}
func addResultHandler(w http.ResponseWriter, r *http.Request) error {
	result := r.FormValue("result")

	if len(result) < 1 {
		w.WriteHeader(200)
		return nil
	}
	client := &http.Client{}
	service, err := pubsub.New(client)
	if err != nil {
		log.Fatalf("Unable to create PubSub service: %v", err)
	}

	_, err = service.Projects.Topics.Create(PubsubTopicID, &pubsub.Topic{}).Do()
	if err != nil {
		log.Fatalf("createTopic Create().Do() failed: %v", err)
	}

	pubsubMessage := &pubsub.PubsubMessage{
		Data: base64.StdEncoding.EncodeToString([]byte(result)),
	}
	publishRequest := &pubsub.PublishRequest{
		Messages: []*pubsub.PubsubMessage{pubsubMessage},
	}
	if _, err := service.Projects.Topics.Publish(PubsubTopicID, publishRequest).Do(); err != nil {
		log.Fatalf("addResultHandler Publish().Do() failed: %v", err)
	}

	fmt.Fprint(w, "Published a message to the topic")

	w.WriteHeader(200)
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
