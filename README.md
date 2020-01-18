# chromepdcv

... allows you to find elements in a website or regions rendered within a canvas and click them.
You can also just get the position or the node(s) at that position.

![alt text](example_match.png "Canvas sample")


### Install
```bash
# opencv: stable 4.2.0 (bottled)
brew install opencv
```
alternative for linux look to the [gocv Dockerfile](https://github.com/hybridgroup/gocv/blob/master/Dockerfile) how to build it right.

```bash
go get -u github.com/Dexus/chromedpcv
```


### Try yourself

```bash
cd example
# install chrome or chromium browser
go build . && ./example
```

##### Documentation
[godoc](https://godoc.org/github.com/Dexus/chromedpcv)

##### Known issues
 - the search image **must** match the real size to get correct position