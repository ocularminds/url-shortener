package shortner

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestFind(t *testing.T) {
	shortlink := GetRandom()
	link, err := Persist(ShortLink{Shortened: shortlink, Original: "http://google.com", Created: time.Now(), Hits: 0})
	if err != nil {
		fmt.Println("error persisting data ", err)
		t.Fail()
	}
	if link.Shortened != shortlink {
		fmt.Println("retrieved short " + link.Shortened + " is not the same as source " + shortlink)
		t.Fail()
	}
	link2, err := Find(shortlink)
	if err != nil {
		fmt.Println("cannot find link", err)
		t.Fail()
	}
	if link2.Shortened != shortlink {
		fmt.Println("retrieved short " + link2.Shortened + " is not the same as source " + shortlink)
		t.Fail()
	}
}

func TestFindWithUnknownUrl(t *testing.T) {
	shortlink := GetRandom()
	Persist(ShortLink{Shortened: shortlink, Original: "http://google.com/search", Created: time.Now(), Hits: 0})
	link2, err := Find("Odumegwu")
	if err != nil {
		fmt.Println("cannot find link", err)
		t.Fail()
	}
	if link2.Shortened == shortlink {
		t.Fail()
	}
}

func TestPersist(t *testing.T) {
	shortlink := GetRandom()
	link, err := Persist(ShortLink{Shortened: shortlink, Original: "http://google.com?s=" + shortlink, Created: time.Now(), Hits: 0})
	if err != nil || len(strings.Split(link.Shortened, "")) < 8 {
		fmt.Println("error persisting data ", err)
		t.Fail()
	}
}

func TestUpdateCount(t *testing.T) {
	shortlink := GetRandom()
	longURL := "https://www.zoom.com?access=" + shortlink + GetRandom() + GetRandom()
	Persist(ShortLink{Shortened: shortlink, Original: longURL, Created: time.Now(), Hits: 0})
	UpdateCount(shortlink)
	UpdateCount(shortlink)
	link2, err := Find(shortlink)
	if err != nil {
		fmt.Println("cannot find link", err)
		t.Fail()
	}
	if link2.Hits != 2 {
		fmt.Println(link2.Hits, " hits but expect 2")
	}
}

func TestFindOriginalLink(t *testing.T) {
	shortlink := GetRandom()
	longURL := "https://www.zoom.com?access=" + shortlink + GetRandom() + GetRandom()
	Persist(ShortLink{Shortened: shortlink, Original: longURL, Created: time.Now(), Hits: 0})
	link2, err := FindByOriginalLink(longURL)
	if err != nil {
		fmt.Println("cannot find link", err)
		t.Fail()
	}
	if link2.Original != longURL {
		fmt.Println("TestFindOriginalLink expects ", longURL, " but was ", link2.Original)
		t.Fail()
	}
}
