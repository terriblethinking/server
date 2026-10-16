package server

import (
	"fmt"
	"log/slog"
	"net/http"

	"context"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	bifrost "github.com/maximhq/bifrost/core"
	"github.com/maximhq/bifrost/core/schemas"
	"github.com/terriblethinking/engine/agent"
)

type Server struct {
	logger *slog.Logger
	client *bifrost.Bifrost
}

type UserRequest struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	ModelID *string `json:"model_id"`
	ProviderID *schemas.ModelProvider `json:"provider_id"`
}

type StreamDone struct {
	State string `json:"state"`
}

func NewServer(logger *slog.Logger, client *bifrost.Bifrost) *Server {
	return &Server{
		logger: logger.With("component", "server"),
		client: client,
	}
}

func (s *Server) ServeWebsocket(w http.ResponseWriter, r *http.Request) {

	s.logger.Info("starting ws")

	http.HandleFunc("/chat/ws", WSWrapper(*s.client, s.logger))

	s.logger.Info("listening on port: 6742")

	err := http.ListenAndServe(":6742", nil)

	if err != nil {
		s.logger.Error(err.Error())
	}

}

func WSWrapper(client bifrost.Bifrost, logger *slog.Logger) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		c, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}

		defer c.CloseNow()

		connLogger := logger.With(
			"remote", r.RemoteAddr,
		)

		connLogger.Info("websocket connection established")

		ctx := context.Background()

		var request UserRequest;
		err = wsjson.Read(ctx, c, &request)


		// Here we initialize the agent. Notice that two critical fields, ProviderID
		// and ModelID, are missing. We will add these later on in the chat loop.

		agent := agent.Agent{
			Client:   client,
			Messages: []schemas.ChatMessage{},
		}

		for {

			// Here we serialize the user request (which has hopefully come
			// in a JSON format) and check whether there was an error.

			var request UserRequest
			err = wsjson.Read(ctx, c, &request)

			if err != nil {
				connLogger.Error("error with reading user request: ", err)
				return
			}

			// If inside the request the model and provider are not empty, we have to
			// update the new settings inside the agent.

			if request.ModelID != nil && request.ProviderID != nil {

				// We first get the model and provider identifiers from the request.

				modelID := request.ModelID
				providerID := request.ProviderID

				connLogger.Info(fmt.Sprintf("updating agent provider/model with \"%s/%s\"", providerID, modelID))

				// We then simply update the fields in our var that holds the agent.

				agent.Provider = *providerID
				agent.Model = *modelID

			}

			switch request.Type {
			case "chat":

				var prompt string = request.Message

				// Set off the main agent loop.

				streamCh := agent.RunAsync(prompt)

				for {
					chunk, ok := <-streamCh

					// A !ok would mean that the channel has finished sending, which
					// would in turn mean that the model has finished replying.

					if !ok {

						err = wsjson.Write(ctx, c, StreamDone{
							State: "done",
						})

						if err != nil {

							connLogger.Error(fmt.Sprintf("failed to send stream done message: %s", err.Error()))
							return

						}

					} else {

						// Otherwise, we simply return the chunk for the frontend
						// to deal with.

						connLogger.Info(fmt.Sprintf("sending message chunk to %s", r.RemoteAddr))

						err = wsjson.Write(ctx, c, chunk)

						if err != nil {

							connLogger.Error(fmt.Sprintf("failed to send chunk: %s", err.Error()))
							return

						}
					}
				}

			case "ping":

				// handle ping

			default:

				connLogger.Info("unknown type of request recieved")

			}
		}
	}
}

