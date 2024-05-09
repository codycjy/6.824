#!/bin/bash

# 定义一个循环，最多执行 10 次
for i in {1..100}; do
    echo "Running iteration $i"
    echo "_________________________"
    go test -race -run "2A"
    if [ $? -ne 0 ]; then  # 如果上一个命令的退出状态不是 0（表示有错误发生）
        echo "Test failed on iteration $i"
        break  # 停止循环
    fi
done
