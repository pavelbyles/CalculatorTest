package hello

import (
	"fmt"
	"golang.org/x/net/context"
	"google.golang.org/appengine/log"
	"net/http"

	"cloud.google.com/go/pubsub"
)

var (
	PubsubClient *pubsub.Client
)

const PubsubTopicID = "projects/calculator-test-182623/topics/calcfinished"

func init() {
	http.HandleFunc("/", handler)
}

func handler(w http.ResponseWriter, r *http.Request) {
	client, err = configurePubSub("calculator-test-182623")
	log.Infof("Created pubsub topic")
	
	fmt.Fprint(w, "Hello, from http / http handler!")
}

func configurePubsub(projectID string) (*pubsub.Client, error) {
	ctx := context.Background()
	client, err := pubsub.NewClient(ctx, projectID)
	topic, _ := PubsubClient.CreateTopic(ctx, PubsubTopicID)
	if err != nil {
		return nil, err
	}

	// Create the topic if it doesn't exist.
	if exists, err := client.Topic(PubsubTopicID).Exists(ctx); err != nil {
		return nil, err
	} else if !exists {
		if _, err := client.CreateTopic(ctx, PubsubTopicID); err != nil {
			return nil, err
		}
	}
	return client, nil
}

func subscribe() {
	ctx := context.Background()
	err := subscription.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
		var id int64
		if err := json.Unmarshal(msg.Data, &id); err != nil {
			log.Printf("could not decode message data: %#v", msg)
			msg.Ack()
			return
		}

		log.Printf("[ID %d] Processing.", id)
		if err := update(id); err != nil {
			log.Printf("[ID %d] could not update: %v", id, err)
			msg.Nack()
			return
		}

		countMu.Lock()
		count++
		countMu.Unlock()

		msg.Ack()
		log.Printf("[ID %d] ACK", id)
	})
	if err != nil {
		log.Fatal(err)
	}
}
