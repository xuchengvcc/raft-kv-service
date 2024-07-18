#!/bin/bash

# 脚本参数
WORKERS=1
ITERATIONS=1
OUTPUT_DIR="./test_logs"
RACE=false
TIMING=false
VERBOSE=0
CPU_PROFILE=""
TEST_FUNC=""

# 解析命令行参数
while [[ "$#" -gt 0 ]]; do
    case $1 in
        -w|--workers) WORKERS="$2"; shift ;;
        -i|--iterations) ITERATIONS="$2"; shift ;;
        -o|--output) OUTPUT_DIR="$2"; shift ;;
        -r|--race) RACE=true ;;
        -t|--timing) TIMING=true ;;
        -v|--verbose) VERBOSE=$((VERBOSE + 1)) ;;
        -cp|--cpuProfile) CPU_PROFILE="$2"; shift ;;
        -f|--func) TEST_FUNC="$2"; shift ;;
        *) echo "Unknown parameter passed: $1"; exit 1 ;;
    esac
    shift
done

# 创建输出目录
mkdir -p "${OUTPUT_DIR}"

# 日志文件
LOG_FILE="${OUTPUT_DIR}/test.log"

# 清空日志文件
> "${LOG_FILE}"

# 执行测试命令
CMD="go test -v ./... -count=${ITERATIONS}"
if [ "$RACE" = true ]; then
    CMD+=" -race"
fi
if [ -n "$CPU_PROFILE" ]; then
    CMD+=" -cpuprofile ${CPU_PROFILE}"
fi
if [ -n "$TEST_FUNC" ]; then
    CMD+=" -run=${TEST_FUNC}"
fi

# 运行测试并将输出重定向到日志文件
echo "Running tests with command: ${CMD}"
${CMD} 2>&1 | tee "${LOG_FILE}"

# 打印测试完成消息
echo "Tests completed. Logs are saved in ${LOG_FILE}"