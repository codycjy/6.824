#!/bin/bash

# 定义一个循环，最多执行 10 次
docker build -t raft . ||exit
for i in {1..10}; do
	echo "Creating docker $i"
	docker run -d --name raft-docker-$i raft
done

