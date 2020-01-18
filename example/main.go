package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"strings"
	"time"

	"github.com/Dexus/chromedpcv"
	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

func main() {

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.DisableGPU,
		//chromedp.Flag("headless", false),
	)

	allocCtx, cancelAll := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancelAll()

	// also set up a custom logger
	taskCtx, cancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(func(s string, i ...interface{}) {
		if strings.Contains(fmt.Sprint(s, i), "chromedpcv") {
			fmt.Println(s, i)
		}
	}), chromedp.WithErrorf(func(s string, i ...interface{}) {
		if strings.Contains(fmt.Sprint(s, i), "chromedpcv") {
			fmt.Println(s, i)
		}
	}))
	defer cancel()

	// create new chromedpcv instance (default config will be set)
	chromecv := chromedpcv.New()

	// save screenshot and mark the image that has been found within
	chromecv.TemplateMatchMarkedScreenShotFilePath = "match.png"

	var imageSearchPosition chromedpcv.BrowserWindowPosition
	_ = imageSearchPosition

	// a smaller viewport speeds up this test
	if err := chromedp.Run(taskCtx, chromedp.EmulateViewport(2048, 1024)); err != nil {
		log.Fatal(err)
	}
	// run task list
	fmt.Println("Run Tasks")
	err := chromedp.Run(taskCtx, chromedp.Tasks{
		chromedp.Navigate("https://jsfiddle.net/hmyrtq9u/5/"),
		WaitSeconds(5),
	})
	if err != nil {
		log.Fatal(err)
	}

	err = chromedp.Run(taskCtx, chromedp.Tasks{
		chromecv.PositionWhereScreenLooksLike("./search_right_black_rect.png", &imageSearchPosition),

		// Don't make this mistake both values will be 0
		DebugPosition(imageSearchPosition.X, imageSearchPosition.Y),
		// MouseClickXY won't work
		// Use pointers
		DebugPositionPointer(&imageSearchPosition),
		chromecv.MouseClickAtPosition(&imageSearchPosition),

		// or search for lookalike region and click it immediately
		chromecv.MouseClickWhereScreenLooksLike("./search_right_black_rect.png"),
	})
	if err != nil {
		log.Fatal(err)
	}

	var buf []byte

	// capture entire browser viewport, returning png with quality=90
	if err := chromedp.Run(taskCtx, fullScreenshotOnly(90, &buf)); err != nil {
		log.Fatal(err)
	}

	if err := ioutil.WriteFile("fullScreenshot.jpeg", buf, 0644); err != nil {
		log.Fatal(err)
	}

	fmt.Println("let the amazing mouse click breath a little")
	time.Sleep(10 * time.Second)
	fmt.Println("take a look at match.png")

	fmt.Println("Good night!")
}

func DebugPosition(x, y float64) chromedp.Action {
	return chromedp.ActionFunc(func(ctxt context.Context) error {
		fmt.Println("values: x: ", x, " y: ", y)
		return nil
	})
}

func DebugPositionPointer(position *chromedpcv.BrowserWindowPosition) chromedp.Action {
	return chromedp.ActionFunc(func(ctxt context.Context) error {
		fmt.Println("pointer: x: ", position.X, " y: ", position.Y)
		return nil
	})
}

func WaitSeconds(num int64) chromedp.Action {
	return chromedp.ActionFunc(func(ctxt context.Context) error {
		fmt.Print("Waiting ", num, " seconds ...")
		time.Sleep(time.Duration(num) * time.Second)
		fmt.Println("Done.")
		return nil
	})
}

// fullScreenshotOnly takes a screenshot of the entire browser viewport.
//
// Liberally copied from puppeteer's source.
//
// Note: this will override the viewport emulation settings.
func fullScreenshotOnly(quality int64, res *[]byte) chromedp.Tasks {
	return chromedp.Tasks{
		chromedp.Sleep(2 * time.Second),
		chromedp.ActionFunc(func(ctx context.Context) error {
			// get layout metrics
			_, _, contentSize, err := page.GetLayoutMetrics().Do(ctx)
			if err != nil {
				return err
			}

			width, height := int64(math.Ceil(contentSize.Width)), int64(math.Ceil(contentSize.Height))

			// force viewport emulation
			err = emulation.SetDeviceMetricsOverride(width, height, 1, false).
				WithScreenOrientation(&emulation.ScreenOrientation{
					Type:  emulation.OrientationTypePortraitPrimary,
					Angle: 0,
				}).
				Do(ctx)
			if err != nil {
				return err
			}

			// capture screenshot
			*res, err = page.CaptureScreenshot().
				WithFormat(page.CaptureScreenshotFormatJpeg).
				WithQuality(quality).
				WithClip(&page.Viewport{
					X:      contentSize.X,
					Y:      contentSize.Y,
					Width:  contentSize.Width,
					Height: contentSize.Height,
					Scale:  1,
				}).Do(ctx)
			if err != nil {
				return err
			}
			return nil
		}),
	}
}
