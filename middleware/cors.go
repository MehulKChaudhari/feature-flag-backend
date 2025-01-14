package middleware

import (
	"log"
	"net/http"
	"regexp"

	"github.com/aws/aws-lambda-go/events"
)

func CORSMiddleware() func(req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return func(req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		log.Println("Received Headers:", req.Headers)
		origin, exists := req.Headers["Origin"]
		if !exists {
			return events.APIGatewayProxyResponse{
				StatusCode: http.StatusForbidden,
				Body:       "CORS policy: No origin header found.",
			}, nil
		}

		rdsPattern := `^https?://([a-zA-Z0-9-]+\.)*realdevsquad\.com$`
		isRDSDomain, err := regexp.MatchString(rdsPattern, origin)
		if err != nil {
			log.Printf("Error matching RDS domain: %v", err)
		}

		if isRDSDomain {

			if req.HTTPMethod == "OPTIONS" {
				return events.APIGatewayProxyResponse{
					StatusCode: http.StatusOK,
					Headers: map[string]string{
						"Access-Control-Allow-Origin":      origin,
						"Access-Control-Allow-Methods":     "GET, POST, PUT, DELETE, OPTIONS",
						"Access-Control-Allow-Headers":     "Authorization, Content-Type, Cache-Control",
						"Access-Control-Allow-Credentials": "true",
						"Access-Control-Expose-Headers":    "Set-Cookie",
						"Vary":                             "Origin",
					},
					Body: "",
				}, nil
			}

			return events.APIGatewayProxyResponse{
				StatusCode: http.StatusOK,
				Headers: map[string]string{
					"Access-Control-Allow-Origin":      origin,
					"Access-Control-Allow-Methods":     "GET, POST, PUT, DELETE, OPTIONS",
					"Access-Control-Allow-Headers":     "Authorization, Content-Type, Cache-Control",
					"Access-Control-Allow-Credentials": "true",
					"Access-Control-Expose-Headers":    "Set-Cookie",
					"Vary":                             "Origin",
				},
			}, nil
		}

		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusForbidden,
			Body:       "CORS policy does not allow access from this origin.",
		}, nil
	}
}
