package server

import (
	"context"
	"encoding/json"
	"fmt"
	service_ext_proc_v3 "github.com/envoyproxy/go-control-plane/envoy/service/ext_proc/v3"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

type AlbumResponse struct {
	Id     int    `json:"id"`
	UserId int    `json:"userId"`
	Title  string `json:"title"`
}

type PostResponse struct {
	Id     int    `json:"id"`
	UserId int    `json:"userId"`
	Title  string `json:"title"`
	Body   string `json:"body"`
}

type AggregatedData struct {
	Albums []AlbumResponse `json:"albums"`
	Posts  []PostResponse  `json:"posts"`
}

type Server struct {
	Log *zap.Logger
}

type HealthServer struct {
	Log *zap.Logger
}

func (s *HealthServer) Check(ctx context.Context, in *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	s.Log.Debug("received health check request", zap.String("service", in.String()))
	return &grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING}, nil
}

func (s *HealthServer) Watch(in *grpc_health_v1.HealthCheckRequest, srv grpc_health_v1.Health_WatchServer) error {
	return status.Error(codes.Unimplemented, "watch is not implemented")
}

func (s *Server) Process(srv service_ext_proc_v3.ExternalProcessor_ProcessServer) error {
	ctx := srv.Context()
	for {
		select {
		case <-ctx.Done():
			s.Log.Debug("context done")
			return ctx.Err()
		default:
		}

		req, err := srv.Recv()
		if err == io.EOF {
			// envoy has closed the stream. Don't return anything and close this stream entirely
			return nil
		}
		if err != nil {
			return status.Errorf(codes.Unknown, "cannot receive stream request: %v", err)
		}

		// build response based on request type
		resp := &service_ext_proc_v3.ProcessingResponse{}
		switch v := req.Request.(type) {
		case *service_ext_proc_v3.ProcessingRequest_RequestHeaders:
			h := req.Request.(*service_ext_proc_v3.ProcessingRequest_RequestHeaders)
			headersResp, err := s.aggregateServices(h.RequestHeaders)
			if err != nil {
				return err
			}
			resp = &service_ext_proc_v3.ProcessingResponse{
				Response: &service_ext_proc_v3.ProcessingResponse_RequestHeaders{
					RequestHeaders: headersResp,
				},
			}

		case *service_ext_proc_v3.ProcessingRequest_RequestBody:
			s.Log.Debug("got RequestBody (not currently implemented)")

		case *service_ext_proc_v3.ProcessingRequest_RequestTrailers:
			s.Log.Debug("got RequestTrailers (not currently implemented)")

		case *service_ext_proc_v3.ProcessingRequest_ResponseHeaders:
			s.Log.Debug("got ResponseHeaders (not currently implemented)")

		case *service_ext_proc_v3.ProcessingRequest_ResponseBody:
			s.Log.Debug("got ResponseBody (not currently implemented)")

		case *service_ext_proc_v3.ProcessingRequest_ResponseTrailers:
			s.Log.Debug("got ResponseTrailers (not currently handled)")

		default:
			s.Log.Error("unknown Request type", zap.Any("v", v))
		}

		// At this point we believe we have created a valid response...
		// note that this is sometimes not the case
		// anyways for now just send it
		s.Log.Debug("sending ProcessingResponse")
		if err := srv.Send(resp); err != nil {
			s.Log.Error("send error", zap.Error(err))
			return err
		}

	}
}

// get the user id from the list of headers
func (s *Server) getUserIdFromHeaders(in *service_ext_proc_v3.HttpHeaders) string {
	for _, n := range in.Headers.Headers {
		if strings.ToLower(n.Key) == "userid" {
			return string(n.RawValue)
		}
	}
	return ""
}

func (s *Server) aggregateServices(in *service_ext_proc_v3.HttpHeaders) (*service_ext_proc_v3.HeadersResponse, error) {
	userIdString := s.getUserIdFromHeaders(in)

	// no instructions were sent, so don't modify anything
	if userIdString == "" {
		return &service_ext_proc_v3.HeadersResponse{}, nil
	}

	// build the response
	resp := &service_ext_proc_v3.HeadersResponse{
		Response: &service_ext_proc_v3.CommonResponse{},
	}

	// required when mutating the body based on a header request
	resp.Response.Status = service_ext_proc_v3.CommonResponse_CONTINUE_AND_REPLACE

	body := s.fetchAggregatedResources(userIdString)
	resp.Response.BodyMutation = &service_ext_proc_v3.BodyMutation{
		Mutation: &service_ext_proc_v3.BodyMutation_Body{
			Body: []byte(body),
		},
	}

	return resp, nil
}

// fetch the albums given a user id
func (s *Server) fetchAlbums(id string, wg *sync.WaitGroup) []AlbumResponse {
	defer wg.Done()

	s.Log.Info("fetching Albums for user", zap.String("user", id))
	url := fmt.Sprintf("https://jsonplaceholder.typicode.com/users/%s/albums", id)

	resp, err := http.Get(url)
	if err != nil {
		s.Log.Fatal("error loading Albums", zap.String("url", url), zap.Error(err))
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			panic(err)
		}
	}(resp.Body)

	var albumResp []AlbumResponse
	decodeErr := json.NewDecoder(resp.Body).Decode(&albumResp)
	if decodeErr != nil {
		s.Log.Fatal("error decoding Albums response", zap.Error(err))
	}

	return albumResp
}

// fetch the posts given a user id
func (s *Server) fetchPosts(id string, wg *sync.WaitGroup) []PostResponse {
	defer wg.Done()

	s.Log.Info("fetching Posts for user", zap.String("user", id))
	url := fmt.Sprintf("https://jsonplaceholder.typicode.com/users/%s/posts", id)

	resp, err := http.Get(url)
	if err != nil {
		s.Log.Fatal("error loading Posts", zap.String("url", url), zap.Error(err))
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			panic(err)
		}
	}(resp.Body)

	var postResp []PostResponse
	decodeErr := json.NewDecoder(resp.Body).Decode(&postResp)
	if decodeErr != nil {
		s.Log.Error("error decoding Posts response", zap.Error(err))
	}
	return postResp
}

func (s *Server) fetchAggregatedResources(id string) string {
	start := time.Now()

	var wg sync.WaitGroup

	var albumsResp []AlbumResponse
	var postsResp []PostResponse

	wg.Add(1)
	go func(id string) {
		albumsResp = s.fetchAlbums(id, &wg)
	}(id)

	wg.Add(1)
	go func(id string) {
		postsResp = s.fetchPosts(id, &wg)
	}(id)

	wg.Wait()

	end := time.Now()
	duration := end.Sub(start)
	s.Log.Info("fetching took", zap.Duration("duration", duration))

	aggregatedData := AggregatedData{}
	aggregatedData.Albums = albumsResp
	aggregatedData.Posts = postsResp

	data, err := json.Marshal(aggregatedData)
	if err != nil {
		s.Log.Error("error marshalling aggregated data", zap.Error(err))
		return ""
	}

	return string(data)
}
