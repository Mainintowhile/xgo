```
first of all thanks creazy-max
i'm hurry to change the code to fit my case;
my code structure:
    - [ ] go.mod
    - [ ] src
        - [ ] portals ------------------->
            - [ ] api -------------------> project1:main1.go
            - [ ] console ---------------> project2:main2.go


go mod situation:
xgo command: 	xgo -go 1.15.2 -mod true -targets $TARGETS -buildDir $buildDir -goPath $GOPATH -goproxy $GOPROXY -out $out -pkg $pkg .(project absolut path)
the convert docker command: docker run --rm -v xx(act:$buildDir):/build -v /tmp/xgo-cache(not used):/deps-cache:ro -e REPO_REMOTE=
-e REPO_BRANCH= -e PACK=/src/portals/lvms -e DEPS= -e ARGS= -e OUT=xx($pkg) -e FLAG_V=false -e FLAG_X=false -e FLAG_RACE=false
-e FLAG_TAGS= -e FLAG_LDFLAGS= -e FLAG_BUILDMODE=default -e TARGETS=linux/amd64 -e GOPROXY=https://goproxy.cn
-v xx($GOPATH):/cache -e GOPATH=/cache -e GO111MODULE=on -v xx($buildDir):/source crazymax/xgo:1.15.2 .(project absolut path)

not go mod situation:
xgo -go 1.15.2 -mod false -targets $TARGETS -buildDir $buildDir -goPath $GOPATH -goproxy $GOPROXY -out $APPNAME xx(project absolut path)
Docker run --rm -v $buildDir:/build
-v /tmp/xgo-cache:/deps-cache:ro -e REPO_REMOTE= -e REPO_BRANCH= -e PACK= -e DEPS= -e ARGS=
-e OUT=lvms -e FLAG_V=false -e FLAG_X=false -e FLAG_RACE=false -e FLAG_TAGS= -e FLAG_LDFLAGS=
-e FLAG_BUILDMODE=default -e TARGETS=linux/amd64 -e GOPROXY=https://goproxy.cn
-v $GOPATH:/cache -e GOPATH=/cache crazymax/xgo:1.15.2 xx(project absolut path)
```
