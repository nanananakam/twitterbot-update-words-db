#!/usr/bin/env bash

aws s3 cp s3://${AWS_S3_BUCKET}/tweets.tar.xz .
tar Jxvf tweets.tar.xz
/main
tar Jcvf words.tar.xz words.db
aws s3 cp words.tar.xz s3://${AWS_S3_BUCKET}/words.tar.xz