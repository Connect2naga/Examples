package main

import (
	"bytes"
	"context"
	"fmt"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/credentials"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"

	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

var (
	serviceName  = "books"
	collectorURL = "localhost:4317"
	insecure     = "true"
)

func initTracer() func(context.Context) error {

	secureOption := otlptracegrpc.WithTLSCredentials(credentials.NewClientTLSFromCert(nil, ""))
	if len(insecure) > 0 {
		secureOption = otlptracegrpc.WithInsecure()
	}

	exporter, err := otlptrace.New(
		context.Background(),
		otlptracegrpc.NewClient(
			secureOption,
			otlptracegrpc.WithEndpoint(collectorURL),
		),
	)

	if err != nil {
		log.Fatal(err)
	}
	resources, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			attribute.String("service.name", serviceName),
			attribute.String("library.language", "go"),
		),
	)
	if err != nil {
		log.Println("Could not set resources: ", err)
	}

	otel.SetTracerProvider(
		sdktrace.NewTracerProvider(
			sdktrace.WithSampler(sdktrace.AlwaysSample()),
			sdktrace.WithBatcher(exporter),
			sdktrace.WithResource(resources),
		),
	)
	return exporter.Shutdown
}

func main() {

	cleanup := initTracer()
	defer cleanup(context.Background())

	r := gin.Default()
	r.Use(otelgin.Middleware(serviceName))
	// Connect to database

	// Routes
	r.GET("/books", FindBooks)

	// Run the server
	r.Run(":8090")
}

func FindBooks(c *gin.Context) {
	span := trace.SpanFromContext(c.Request.Context())
	span.SetAttributes(attribute.String("controller", "books"))
	span.AddEvent("This is a sample event",
		trace.WithAttributes(attribute.Int("pid", 4328), attribute.String("sampleAttribute", "Test")))
	makeCall(c.Request.Context())
	c.JSON(http.StatusOK, gin.H{"data": "books"})
}

func makeCall(ctx context.Context) {
	requestURL := "http://localhost:9090/store"

	jsonBody := []byte(`{
    "username": "string",
    "password": "string"
}`)
	bodyReader := bytes.NewReader(jsonBody)
	httpClient := &http.Client{}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, bodyReader)
	if err != nil {
		fmt.Printf("client: could not create request: %s\n", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := httpClient.Do(req)
	if err != nil {
		fmt.Printf("client: error making http request: %s\n", err)
		os.Exit(1)
	}
	fmt.Printf("%v", res)

}
