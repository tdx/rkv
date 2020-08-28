#! /bin/bash
# Make sure you grab the latest version
#curl -OL https://github.com/google/protobuf/releases/download/v3.6.1/protoc-3.6.1-linux-x86_64.zip
curl -OL https://github.com/protocolbuffers/protobuf/releases/download/v3.13.0/protoc-3.13.0-linux-x86_64.zip

# Unzip
unzip protoc-3.13.0-linux-x86_64.zip -d protoc3

# Move protoc to /usr/local/bin/
sudo cp -r protoc3/bin/* /usr/local/bin/

# Move protoc3/include to /usr/local/include/
sudo cp -r protoc3/include/* /usr/local/include/

# Optional: change owner
sudo chown $USER /usr/local/bin/protoc
sudo chown -R $USER /usr/local/include/google

sudo ldconfig

go get github.com/gogo/protobuf/protoc-gen-gofast
go get github.com/gogo/protobuf/proto
go get github.com/gogo/protobuf/protoc-gen-gogofast
go get github.com/gogo/protobuf/protoc-gen-gogofaster
go get github.com/gogo/protobuf/gogoproto
