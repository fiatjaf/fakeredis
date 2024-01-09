package test

import (
	"testing"

	"github.com/fiatjaf/fakeredis"
	"github.com/redis/go-redis/v9"
)

func TestSortedSet(t *testing.T) {
	rds := fakeredis.New()
	rds.ZAdd("letter",
		redis.Z{Score: 1, Member: "a"},
	)
	rds.ZAdd("letter",
		redis.Z{Score: 6, Member: "f"},
		redis.Z{Score: 5, Member: "e"},
		redis.Z{Score: 3, Member: "c"},
		redis.Z{Score: 20, Member: "b"},
	)
	rds.ZAdd("letter",
		redis.Z{Score: 4, Member: "d"},
		redis.Z{Score: 2, Member: "b"},
	)
	if n := rds.ZCard("letter").Val(); n != 6 {
		t.Fatalf("letter zcard failed: %d", n)
	}
	list := rds.ZRangeByScore("letter", nil).Val()
	if len(list) != 6 {
		t.Fatalf("letter len zrangebyscore 1 failed: %v", list)
	}
	if list[1] != "b" || list[2] != "c" || list[5] != "f" {
		t.Fatalf("letter ordering 1 failed: %v", list)
	}

	list = rds.ZRangeByScore("letter", &redis.ZRangeBy{Offset: 2}).Val()
	if len(list) != 4 {
		t.Fatalf("letter len zrangebyscore 2 failed: %v", list)
	}
	list = rds.ZRangeByScore("letter", &redis.ZRangeBy{Offset: 2, Max: "19"}).Val()
	if len(list) != 4 {
		t.Fatalf("letter len zrangebyscore 3 failed: %v", list)
	}
	list = rds.ZRangeByScore("letter", &redis.ZRangeBy{Offset: 2, Max: "+inf", Min: "-inf"}).Val()
	if len(list) != 4 {
		t.Fatalf("letter len zrangebyscore 4 failed: %v", list)
	}
	list = rds.ZRangeByScore("letter", &redis.ZRangeBy{Offset: 2, Count: 2, Max: "+inf", Min: "-inf"}).Val()
	if len(list) != 2 {
		t.Fatalf("letter len zrangebyscore 5 failed: %v", list)
	}
	if list[0] != "c" || list[1] != "d" {
		t.Fatalf("letter ordering 5 failed: %v", list)
	}
}

func TestSortedSet2(t *testing.T) {
	rds := fakeredis.New()
	rds.ZAdd("number",
		redis.Z{Score: 1, Member: "1"},
		redis.Z{Score: 2, Member: "2"},
		redis.Z{Score: 1, Member: "1"},
		redis.Z{Score: 2, Member: "2"},
		redis.Z{Score: 3, Member: "3"},
		redis.Z{Score: 4, Member: "4"},
		redis.Z{Score: 5, Member: "5"},
		redis.Z{Score: 6, Member: "6"},
	)
	if n := rds.ZCard("number").Val(); n != 6 {
		t.Fatalf("zcard number failed: %d", n)
	}

	list := rds.ZRangeByScore("number", &redis.ZRangeBy{Offset: 2}).Val()
	if len(list) != 4 {
		t.Fatalf("number len zrangebyscore 1 failed: %v", list)
	}

	rds.ZAdd("number", redis.Z{Score: 0, Member: "0"})
	list = rds.ZRangeByScore("number", &redis.ZRangeBy{Count: 28}).Val()
	if len(list) != 7 {
		t.Fatalf("number len zrangebyscore 2 failed: %v", list)
	}
	if list[0] != "0" {
		t.Fatalf("number ordering 2 failed: %v", list)
	}

	rds.ZAdd("number", redis.Z{Score: 5, Member: "2"})
	list = rds.ZRangeByScore("number", nil).Val()
	if len(list) != 7 {
		t.Fatalf("number len zrangebyscore 3 failed: %v", list)
	}
	if list[0] != "0" || list[1] != "1" || list[4] != "2" || list[5] != "5" {
		t.Fatalf("number ordering 3 failed: %v", list)
	}

	rds.ZAdd("number", redis.Z{Score: 100, Member: "1"})
	list = rds.ZRangeByScore("number", nil).Val()
	if len(list) != 7 {
		t.Fatalf("number len zrangebyscore 4 failed: %v", list)
	}
	if list[0] != "0" || list[1] != "3" || list[3] != "2" || list[6] != "1" {
		t.Fatalf("number ordering 4 failed: %v", list)
	}
}
