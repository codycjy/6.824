FROM golang

WORKDIR /app

# Set Go environment variables
RUN go env -w GO111MODULE=on
RUN go env -w GOPROXY=https://goproxy.cn,direct
RUN go env -w CGO_ENABLED=1

# Copy everything from the current directory to the working directory in the container
COPY . .

# Change permissions for the script
RUN chmod +x /app/src/raft/loop_test.sh

# Set the entrypoint to execute the script

ENTRYPOINT ["sh", "-c", "cd /app/src/raft && bash loop_test.sh"]

