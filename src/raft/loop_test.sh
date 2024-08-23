#!/bin/bash

# 定义测试模式变量
TEST_PATTERN="TestBasicAgree2B|TestRPCBytes2B|TestFailAgree2B"

for i in {1..100}; do
    echo "Running iteration $i"
    echo "_________________________"
    
    go test -race -run "$TEST_PATTERN" > test.log
    
    if [ $? -ne 0 ]; then  # 如果上一个命令的退出状态不是 0（表示有错误发生）
        cat test.log
        echo "Test failed on iteration $i"
        break  # 停止循环
    fi
done
