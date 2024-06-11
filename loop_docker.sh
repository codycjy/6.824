#!/bin/bash

# 定义一个循环，最多执行 10 次
docker build -t raft .
for i in {1..16}; do
	echo "Creating docker $i"
	docker run -d --name docker-$i raft
done

