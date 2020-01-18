package chromedpcv

type BrowserWindow struct {
	Width  int64
	Height int64
}

type BrowserWindowPosition struct {
	window *BrowserWindow
	X      float64
	Y      float64
}
