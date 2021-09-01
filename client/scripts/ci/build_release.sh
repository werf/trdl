#!/bin/bash -e

export RELEASE_BUILD_DIR=release-build
export GO111MODULE=on
export CGO_ENABLED=0

go_build_v2() {
    VERSION=$1

    rm -rf $RELEASE_BUILD_DIR/$VERSION
    mkdir -p $RELEASE_BUILD_DIR/$VERSION
    chmod -R 0777 $RELEASE_BUILD_DIR/$VERSION

    for os in linux darwin windows ; do
        for arch in amd64 arm64 ; do
            if [ "$os" == "windows" ] && [ "$arch" == "arm64" ] ; then
                continue
            fi

            outputFile=$RELEASE_BUILD_DIR/$VERSION/$os-$arch/bin/trdl
            if [ "$os" == "windows" ] ; then
                outputFile=$outputFile.exe
            fi

            echo "# Building trdl $VERSION for $os $arch ..."

            GOOS=$os GOARCH=$arch \
              go build -tags "dfrunmount dfssh" -ldflags="-s -w -X github.com/werf/trdl/client/pkg/trdl.Version=$VERSION" \
                       -o $outputFile github.com/werf/trdl/client/cmd/trdl

            echo "# Built $outputFile"
        done
    done
}

VERSION=$1
if [ -z "$VERSION" ] ; then
    echo "Required version argument!" 1>&2
    echo 1>&2
    echo "Usage: $0 VERSION" 1>&2
    exit 1
fi

( go_build_v2 $VERSION ) || ( echo "Failed to build!" 1>&2 && exit 1 )