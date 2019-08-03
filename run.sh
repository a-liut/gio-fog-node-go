docker build --build-arg SRC_ROOT=gio-fog-node-go -f docker/Dockerfile -t gio-fog-node-go:latest .
docker run --rm -it --net host --privileged gio-fog-node-go:latest