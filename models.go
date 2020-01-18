package chromedpcv

// BrowserWindow BrowserWindow
type BrowserWindow struct {
	Width  int64
	Height int64
}

// BrowserWindowPosition BrowserWindowPosition
type BrowserWindowPosition struct {
	window *BrowserWindow
	X      float64
	Y      float64
}
