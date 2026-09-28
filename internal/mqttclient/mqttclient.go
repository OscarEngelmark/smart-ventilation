// Package mqttclient connects a service to the MQTT broker and publishes JSON
// messages on it. Reasoning: project_notes.md §4.
package mqttclient

import (
	"encoding/json"
	"log"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// Connect opens a link to the broker at brokerURL under clientID. The client
// reconnects on its own, and onConnect, unless nil, runs on every connect.
func Connect(brokerURL, clientID string, onConnect mqtt.OnConnectHandler) (mqtt.Client, error) {
	onConnectionLost := func(_ mqtt.Client, err error) {
		log.Printf("connection to broker lost: %v", err)
	}

	opts := mqtt.NewClientOptions().
		AddBroker(brokerURL).
		SetClientID(clientID).
		SetOnConnectHandler(onConnect).
		SetConnectionLostHandler(onConnectionLost)
	client := mqtt.NewClient(opts)
	token := client.Connect()
	token.Wait() // block until the broker has answered
	return client, token.Error()
}

// PublishJSON encodes msg as JSON and publishes it on topic at QoS 1, waiting
// until the broker has taken it. A retained message is kept by the broker and
// handed to any later subscriber.
func PublishJSON(client mqtt.Client, topic string, retained bool, msg any) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	token := client.Publish(topic, 1, retained, payload)
	token.Wait() // block until the broker has taken the message
	return token.Error()
}
