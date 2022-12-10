package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/labstack/echo"
	"github.com/rinser/hw5/feed"
	"github.com/stretchr/testify/assert"
)

var testServer *echo.Echo
var testService *feed.Service

func TestMain(m *testing.M) {
	testServer = echo.New()
	var err error
	testService, err = feed.NewService(
		"test:test@tcp(127.0.0.1:3301)/social_network",
		"localhost:7000",
		"amqp://test:test@localhost:5672/")
	if err != nil {
		log.Fatal(err)
		os.Exit(-1)
	} else {
		defer testService.Cancel()
		scriptBytes, err := os.ReadFile("db.sql")
		if err != nil {
			log.Fatal(err)
			os.Exit(-1)
		}
		scripts := strings.Split(string(scriptBytes), "--")
		// run db schema creation script
		for _, script := range scripts {
			_, err = testService.Db().Exec(string(script))
			if err != nil {
				log.Fatal(err)
				os.Exit(-1)
			}
		}
		exitVal := m.Run()
		os.Exit(exitVal)
	}
}

func TestAddUser(t *testing.T) {
	// Setup
	userJSON := `{"login":"user0"}`
	req := httptest.NewRequest(http.MethodPost, "/user",
		strings.NewReader(userJSON))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := testServer.NewContext(req, rec)
	// Assertions
	if assert.NoError(t, testService.AddUser(c)) {
		assert.Equal(t, http.StatusCreated, rec.Code)
	}
}

func TestAddFollower(t *testing.T) {
	// Setup
	userJSON := `{"login":"user0"}`
	req := httptest.NewRequest(http.MethodPost, "/user",
		strings.NewReader(userJSON))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := testServer.NewContext(req, rec)
	if assert.NoError(t, testService.AddUser(c)) {
		assert.Equal(t, http.StatusCreated, rec.Code)
	}
	userId1, _ := strconv.ParseInt(strings.Trim(rec.Body.String(), "\n"), 10, 64)
	userJSON = `{"login":"user1"}`
	req = httptest.NewRequest(http.MethodPost, "/user",
		strings.NewReader(userJSON))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec = httptest.NewRecorder()
	c = testServer.NewContext(req, rec)
	if assert.NoError(t, testService.AddUser(c)) {
		assert.Equal(t, http.StatusCreated, rec.Code)
	}
	userId2, _ := strconv.ParseInt(strings.Trim(rec.Body.String(), "\n"), 10, 64)
	followerJSON := fmt.Sprintf(`{"userId":%d,"followerId":%d}`,
		userId1, userId2)
	req = httptest.NewRequest(http.MethodPost, "/follower",
		strings.NewReader(followerJSON))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec = httptest.NewRecorder()
	c = testServer.NewContext(req, rec)
	// Assertions
	if assert.NoError(t, testService.AddFollower(c)) {
		assert.Equal(t, http.StatusCreated, rec.Code)
		assert.Equal(t, "true", strings.Trim(rec.Body.String(), "\n"))
	}
}