package main

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/go-kit/kit/endpoint"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	openParenthesis  = "("
	closeParenthesis = ")"
	openCurly        = "{"
	closeCurly       = "}"
	openSquare       = "["
	closeSquare      = "]"
)

// StringService provides operations on strings.
type StringService interface {
	Validate(string) (string, error)
	Fix(string) (string, error)
}

// stringService is a concrete implementation of StringService
type stringService struct{}

func (stringService) Validate(s string) (string, error) {
	if s == "" {
		return "", ErrEmpty
	}
	list := make([]string, 0)
	var err error
	a := strings.Split(s, "")
	for _, r := range a {
		switch r {
		case openParenthesis, openCurly, openSquare:
			list = append(list, r)
		case closeParenthesis:
			list, err = ValidateBracked(list, openParenthesis)
			if err != nil {
				return err.Error(), nil
			}
		case closeCurly:
			list, err = ValidateBracked(list, closeCurly)
			if err != nil {
				return err.Error(), nil
			}
		case closeSquare:
			list, err = ValidateBracked(list, closeSquare)
			if err != nil {
				return err.Error(), nil
			}
		default:
			return "Not Balanced", ErrBadString
		}
	}
	result := len(list)
	if result == 0 {
		return "Balanced", nil
	}
	return "Not Balanced", nil
}
func ValidateBracked(list []string, bracketOpen string) ([]string, error) {
	if list[len(list)-1] != bracketOpen {
		return list, errors.New("Not Balanced")
	}
	list = list[:len(list)-1]
	return list, nil
}

func (stringService) Fix(s string) (string, error) {
	list := make([]string, 0)
	a := strings.Split(s, "")
	result := ""

	for _, r := range a {
		tmp := ""
		switch r {
		case openParenthesis, openCurly, openSquare:
			list = append(list, r)
			result += r
		case closeParenthesis:
			list, tmp = FixBracked(list, openParenthesis, closeParenthesis)
			result += tmp
		case closeCurly:
			list, tmp = FixBracked(list, openCurly, closeCurly)
			result += tmp
		case closeSquare:
			list, tmp = FixBracked(list, openSquare, closeSquare)
			result += tmp
		default:
			return "", ErrBadString
		}
	}
	return result, nil
}
func FixBracked(list []string, bracketOpen string, bracketClose string) ([]string, string) {
	result := ""
	if len(list) > 0 {
		if list[len(list)-1] != bracketOpen {
			list = append(list, bracketOpen)
			result += bracketOpen
		}
		list = list[:len(list)-1]
		result += bracketClose
	} else {
		list = append(list, bracketOpen)
		result += bracketOpen
		result += bracketClose
	}
	return list, result
}

// ErrEmpty is returned when an input string is empty.
var ErrEmpty = errors.New("empty string")
var ErrBadString = errors.New("bad string")

// For each method, we define request and response structs
type validateRequest struct {
	S string `json:"s"`
}

type validateResponse struct {
	V   string `json:"v"`
	Err string `json:"err,omitempty"` // errors don't define JSON marshaling
}

type fixRequest struct {
	S string `json:"s"`
}

type fixResponse struct {
	S   string `json:"v"`
	Err string `json:"err,omitempty"` // errors don't define JSON marshaling
}

// Endpoints are a primary abstraction in go-kit. An endpoint represents a single RPC (method in our service interface)
func makeValidateEndpoint(svc StringService) endpoint.Endpoint {
	return func(_ context.Context, request interface{}) (interface{}, error) {
		opsRequest.Inc()
		req := request.(validateRequest)
		v, err := svc.Validate(req.S)
		if err != nil {
			if err == ErrBadString {
				return validateResponse{req.S, err.Error()}, nil
			}
			return validateResponse{v, err.Error()}, nil
		}
		return validateResponse{v, ""}, nil
	}
}

func makeFixEndpoint(svc StringService) endpoint.Endpoint {
	return func(_ context.Context, request interface{}) (interface{}, error) {
		opsRequest.Inc()
		req := request.(fixRequest)
		v, err := svc.Fix(req.S)
		if err != nil {
			return fixResponse{req.S, err.Error()}, nil
		}

		return fixResponse{v, ""}, nil

	}
}
func main() {
	svc := stringService{}

	validateHandler := httptransport.NewServer(
		makeValidateEndpoint(svc),
		decodeUppercaseRequest,
		encodeResponse,
	)

	countHandler := httptransport.NewServer(
		makeFixEndpoint(svc),
		decodeCountRequest,
		encodeResponse,
	)
	recordMetrics()
	http.Handle("/metrics", promhttp.Handler())
	http.Handle("/validate", validateHandler)
	http.Handle("/fix", countHandler)
	log.Fatal(http.ListenAndServe(":8000", nil))
}
func recordMetrics() {
	go func() {
		for {
			opsProcessed.Inc()
			time.Sleep(2 * time.Second)
		}
	}()
}

var (
	opsProcessed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "myapp_processed_ops_total",
		Help: "The total number of processed events",
	})
	opsRequest = promauto.NewCounter(prometheus.CounterOpts{
		Name: "myapp_total_requests",
		Help: "The total number requests",
	})
)

func decodeUppercaseRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request validateRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return nil, err
	}
	return request, nil
}

func decodeCountRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request fixRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return nil, err
	}
	return request, nil

}

func encodeResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	return json.NewEncoder(w).Encode(response)
}
