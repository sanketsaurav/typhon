package server

import (
	"fmt"
	"os"
	"strings"
	"time"

	log "github.com/cihub/seelog"
	"github.com/golang/protobuf/proto"
	"github.com/streadway/amqp"

	"github.com/vinceprignano/bunny/rabbit"
)

type AMQPServer struct {
	// this is the routing key prefix for all endpoints
	ServiceName        string
	ServiceDescription string
	endpointRegistry   *EndpointRegistry
	connection         *rabbit.RabbitConnection
}

func NewAMQPServer() Server {
	return &AMQPServer{
		endpointRegistry: NewEndpointRegistry(),
		connection:       rabbit.NewRabbitConnection(),
	}
}

func (s *AMQPServer) Name() string {
	if s == nil {
		return ""
	}
	return s.ServiceName
}

func (s *AMQPServer) Description() string {
	if s == nil {
		return ""
	}
	return s.ServiceDescription
}

func (s *AMQPServer) Initialise(c *Config) {
	s.ServiceName = c.Name
	s.ServiceDescription = c.Description
}

func (s *AMQPServer) RegisterEndpoint(endpoint Endpoint) {
	s.endpointRegistry.Register(endpoint)
}

func (s *AMQPServer) DeregisterEndpoint(endpointName string) {
	s.endpointRegistry.Deregister(endpointName)
}

// Run the server, connecting to our transport and serving requests
func (s *AMQPServer) Run() {

	// Connect to AMQP
	select {
	case <-s.connection.Init():
		log.Info("[Server] Connected to RabbitMQ")
	case <-time.After(10 * time.Second):
		log.Critical("[Server] Failed to connect to RabbitMQ")
		os.Exit(1)
	}

	// Get a delivery channel from the connection
	log.Infof("[Server] Listening for deliveries on %s.#", s.ServiceName)
	deliveries, err := s.connection.Consume(s.ServiceName)
	if err != nil {
		log.Criticalf("[Server] [%s] Failed to consume from Rabbit", s.ServiceName)
	}

	// Handle deliveries
	for req := range deliveries {
		log.Infof("[Server] [%s] Received new delivery", s.ServiceName)
		go s.handleRequest(req)
	}

	log.Infof("Exiting")
	log.Flush()
}

func (s *AMQPServer) handleRequest(delivery amqp.Delivery) {

	endpointName := strings.Replace(delivery.RoutingKey, fmt.Sprintf("%s.", s.ServiceName), "", -1)
	endpoint := s.endpointRegistry.Get(endpointName)
	if endpoint == nil {
		log.Error("[Server] Endpoint not found, cannot handle request")
		return
	}
	req := NewAMQPRequest(&delivery)
	rsp, err := endpoint.HandleRequest(req)
	if err != nil {
		log.Errorf("[Server] Endpoint %s returned an error", endpointName)
		log.Error(err.Error())
	}
	body, err := proto.Marshal(rsp)
	if err != nil {
		log.Errorf("[Server] Failed to marshal response")
	}
	msg := amqp.Publishing{
		CorrelationId: delivery.CorrelationId,
		Timestamp:     time.Now().UTC(),
		Body:          body,
	}
	s.connection.Publish("", delivery.ReplyTo, msg)
}
