package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/shogo82148/go-mecab"
	"io"
	"os"
	"os/exec"
	"regexp"
	"time"
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

	svc := s3.New(session.New(), &aws.Config{
		Region: aws.String(os.Getenv("AWS_DEFAULT_REGION")),
	})

	//backup
	_, err := svc.CopyObject(&s3.CopyObjectInput{
		Bucket:     aws.String(os.Getenv("AWS_S3_BUCKET")),
		CopySource: aws.String(os.Getenv("AWS_S3_BUCKET") + "/tweets.tar.xz"),
		Key:        aws.String("backup/tweets" + fmt.Sprint(time.Now().Format("2011-01-31")) + ".tar.xz"),
	})
	if err != nil {
		fmt.Errorf("s3 backup copy failed")
		panic(err)
	}

	s3file, err := svc.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(os.Getenv("AWS_S3_BUCKET")),
		Key:    aws.String("/tweets.tar.xz"),
	})
	if err != nil {
		fmt.Errorf("S3 Get Object failed.\n")
		panic(err)
	}
	defer s3file.Body.Close()

	file, err := os.Create("tweets.tar.xz")
	if err != nil {
		fmt.Errorf("File Create failed.\n")
		panic(err)
	}
	defer file.Close()

	if _, err := io.Copy(file, s3file.Body); err != nil {
		fmt.Errorf("File Write Failed.\n")
		panic(err)
	}

	if output, err := exec.Command("sh", "-c", "tar Jxvf tweets.tar.xz").CombinedOutput(); err != nil {
		fmt.Errorf(string(output))
		fmt.Errorf("File Deflate failed.\n")
		panic(err)
	}

	db, err := gorm.Open("sqlite3", "tweets.db")
	if err != nil {
		fmt.Errorf("Tweets db open failed.\n")
		panic(err)
	}
	defer db.Close()

	db2, err := gorm.Open("sqlite3", "words.db")
	if err != nil {
		fmt.Errorf("Words db open failed.\n")
		panic(err)
	}
	defer db2.Close()
	db2.AutoMigrate(&Words{})

	tagger, err := mecab.New(map[string]string{"output-format-type": "wakati"})
	if err != nil {
		fmt.Errorf("Mecab tagger create failed.\n")
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
				fmt.Errorf("ParseNode failed.\n")
				panic(err)
			}
			word1 := ""
			word2 := ""
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

	if output, err := exec.Command("sh", "-c", "tar Jcvf words.tar.xz words.db").CombinedOutput(); err != nil {
		fmt.Errorf(string(output))
		fmt.Errorf("Words tar failed.\n")
		panic(err)
	}

	file2, err := os.Open("words.tar.xz")
	if err != nil {
		fmt.Errorf("Words tar open failed.\n")
		panic(err)
	}

	if _, err = svc.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(os.Getenv("AWS_S3_BUCKET")),
		Key:    aws.String("/words.tar.xz"),
		Body:   file2,
	}); err != nil {
		fmt.Errorf("Words tar upload failed.\n")
		panic(err)
	}

}
