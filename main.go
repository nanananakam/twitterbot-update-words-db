package main

import (
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/shogo82148/go-mecab"
	"regexp"
)

type Tweet struct {
	TwitterID string `gorm:"unique_index"`
	Tweet     string `gorm:"type:varchar(512)"`
}

type Words struct {
	Word1 string `gorm:"index;type:varchar(512)"`
	Word2 string `gorm:"type:varchar(512)"`
}

func main() {

	db, err := gorm.Open("sqlite3", "tweets.db")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	db2, err := gorm.Open("sqlite3", "words.db")
	if err != nil {
		panic(err)
	}
	defer db2.Close()
	db2.AutoMigrate(&Words{})

	tagger, err := mecab.New(map[string]string{"output-format-type": "wakati"})
	if err != nil {
		panic(err)
	}
	defer tagger.Destroy()

	filterRep := regexp.MustCompile(`(RT|@[^ 　]+|http[^ 　]+|\\)`)
	ngRep := regexp.MustCompile(`(死ね|殺|爆破)`)

	var tweets []Tweet

	db.Find(&tweets)

	tx2 := db2.Begin()

	for _, tweet := range tweets {

		if !ngRep.MatchString(tweet.Tweet) {
			tweetString := filterRep.ReplaceAllString(tweet.Tweet, "")

			node, err := tagger.ParseToNode(tweetString)
			if err != nil {
				panic(err)
			}
			var word1 string
			var word2 string
			for ; node != (mecab.Node{}); node = node.Next() {
				if node.Surface() != "" {
					word2 = node.Surface()
					words := Words{
						Word1: word1,
						Word2: word2,
					}
					tx2.Create(&words)
					word1 = word2
				}
			}
			if word1 != "" {
				word2 = ""
				words := Words{
					Word1: word1,
					Word2: word2,
				}
				tx2.Create(&words)
			}
		}
	}

	tx2.Commit()
}
