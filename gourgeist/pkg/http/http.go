package http

import (
	"context"
	"iv2/gourgeist/pkg/mg"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type httpStore interface {
	mg.GlucoseStore
}

type HttpServer struct {
	Store httpStore
}

func New(s httpStore) *HttpServer {
	hs := &HttpServer{
		Store: s,
	}
	hs.serve()
	return hs
}

func (s *HttpServer) serve() {
	r := gin.Default()

	r.GET("/glucose", func(c *gin.Context) {
		end := c.DefaultQuery("end", "")
		endUnix, err := strconv.Atoi(end)
		if err != nil {
			c.String(http.StatusBadRequest, "expected unix timestamp for end")
			return
		}

		start := c.DefaultQuery("start", "")
		startUnix, err := strconv.Atoi(start)
		if err != nil {
			c.String(http.StatusBadRequest, "expected unix timestamp for start")
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		glucose, err := s.Store.ReadGlucose(ctx, time.Unix(int64(startUnix), 0), time.Unix(int64(endUnix), 0))
		if err != nil {
			c.String(http.StatusInternalServerError, "something went wrong reading glucose: %w", err)
			return
		}

		c.JSON(http.StatusOK, glucose)
	})

	r.Run(":4242")
}
