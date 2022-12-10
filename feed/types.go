package feed

import "time"

type User struct {
	Id    int64  `json:"id"`
	Login string `json:"name"`
}

type Follower struct {
	UserId     int64 `json:"userId"`
	FollowerId int64 `json:"followerId"`
}

type Publication struct {
	Author int64     `json:"from"`
	Text   string    `json:"text"`
	At     time.Time `json:"at"`
}
