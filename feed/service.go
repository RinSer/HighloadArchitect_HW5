package feed

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-redis/redis/v9"
	_ "github.com/go-sql-driver/mysql"
	"github.com/labstack/echo"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Service struct {
	ctx    context.Context
	cancel context.CancelFunc
	db     *sql.DB
	rdb    *redis.Client
	conn   *amqp.Connection
	ch     *amqp.Channel
	queue  amqp.Queue
}

func NewService(
	sqlConnection string,
	redisHost string,
	rabbitConnection string) (*Service, error) {
	var err error
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		if err != nil {
			cancel()
		}
	}()

	// connect to MySQL
	db, err := sql.Open("mysql", sqlConnection)
	if err != nil {
		return nil, err
	}

	// connect to Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisHost,
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	// connect to RabbitMQ
	conn, err := amqp.Dial(rabbitConnection)
	if err != nil {
		return nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}
	queue, err := ch.QueueDeclare(
		"publications", // name
		false,          // durable
		false,          // delete when unused
		false,          // exclusive
		false,          // no-wait
		nil,            // arguments
	)

	return &Service{
		ctx, cancel, db, rdb, conn, ch, queue,
	}, nil
}

func (s *Service) Db() *sql.DB {
	return s.db
}

func (s *Service) Cancel() {
	defer s.ch.Close()
	defer s.conn.Close()
	s.cancel()
}

// API handlers

func (s *Service) AddUser(c echo.Context) (err error) {
	u := new(User)
	err = c.Bind(u)
	if err != nil {
		return
	}
	tx, err := s.db.BeginTx(s.ctx, nil)
	defer func() {
		if err == nil {
			err = tx.Commit()
		} else {
			_ = tx.Rollback()
		}
	}()
	if err != nil {
		return
	}
	_, err = tx.ExecContext(s.ctx,
		`INSERT INTO users (login) values (?);`, u.Login)
	if err != nil {
		return
	}
	row := tx.QueryRowContext(s.ctx, `SELECT LAST_INSERT_ID();`)
	row.Scan(&u.Id)
	return c.JSON(http.StatusCreated, u.Id)
}

func (s *Service) AddFollower(c echo.Context) (err error) {
	f := new(Follower)
	err = c.Bind(f)
	if err != nil {
		return
	}
	tag, err := s.db.ExecContext(s.ctx,
		`INSERT INTO followers (userId, followerId) values (?, ?);`,
		f.UserId, f.FollowerId)
	if err != nil {
		return
	}
	rowsAffected, err := tag.RowsAffected()
	if err != nil {
		return
	}
	redisKey := fmt.Sprintf("%dfollows", f.FollowerId)
	s.rdb.LPush(s.ctx, redisKey, f.UserId)
	return c.JSON(http.StatusCreated, rowsAffected == 1)
}

func (s *Service) AddPublication(c echo.Context) (err error) {
	return c.JSON(http.StatusCreated, nil)
}

func (s *Service) GetFeed(c echo.Context) (err error) {
	userId, err := strconv.ParseInt(c.Param("userId"), 10, 64)
	if err != nil {
		return
	}
	return c.JSON(http.StatusOK, userId)
}
