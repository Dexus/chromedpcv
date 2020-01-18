package chromedpcv

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"log"
	"os"
	"strings"

	"github.com/Dexus/chromedpcv/javascript"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
	"github.com/pkg/errors"
	"gocv.io/x/gocv"
)

// ChromeDpCV struct
type ChromeDpCV struct {
	// TemplateMatchMarkedScreenShotFilePath allows to to keep an image of the screenshot where the searched region is marked, this helps with debugging
	TemplateMatchMarkedScreenShotFilePath string
	TemplateMatchMode                     gocv.TemplateMatchMode
	IMReadFlag                            gocv.IMReadFlag
	// Debug show errors messages with stack
	Debug bool
}

// New create a ChromeDpCV object
func New() *ChromeDpCV {
	return &ChromeDpCV{
		Debug:                                 true,
		TemplateMatchMarkedScreenShotFilePath: "",
		TemplateMatchMode:                     gocv.TmCcoeffNormed,
		IMReadFlag:                            gocv.IMReadColor,
	}
}

// MouseClickWhereScreenLooksLike MouseClickWhereScreenLooksLike
func (c *ChromeDpCV) MouseClickWhereScreenLooksLike(targetImagePath string, opts ...chromedp.MouseOption) chromedp.Action {
	return chromedp.ActionFunc(func(ctxt context.Context) error {
		var position BrowserWindowPosition
		err := c.PositionWhereScreenLooksLike(targetImagePath, &position).Do(ctxt)
		if err != nil {
			return c.errorDebug(err)
		}
		err = chromedp.MouseClickXY(position.X, position.Y, opts...).Do(ctxt)
		if err != nil {
			return c.errorDebug(err)
		}
		return nil
	})
}

// MouseClickAtPosition MouseClickAtPosition
func (c *ChromeDpCV) MouseClickAtPosition(position *BrowserWindowPosition, opts ...chromedp.MouseOption) chromedp.Action {
	return chromedp.ActionFunc(func(ctxt context.Context) error {
		err := chromedp.MouseClickXY(position.X, position.Y, opts...).Do(ctxt)
		if err != nil {
			return c.errorDebug(err)
		}
		return nil
	})
}

// NodesWhereScreenLooksLike NodesWhereScreenLooksLike
func (c *ChromeDpCV) NodesWhereScreenLooksLike(targetImagePath string, resultNodes *[]*cdp.Node) chromedp.Action {
	return chromedp.ActionFunc(func(ctxt context.Context) error {
		var position BrowserWindowPosition
		err := c.PositionWhereScreenLooksLike(targetImagePath, &position).Do(ctxt)
		if err != nil {
			return c.errorDebug(err)
		}
		err = c.NodesAtPosition(&position, resultNodes).Do(ctxt)
		if err != nil {
			return c.errorDebug(err)
		}
		return nil
	})
}

// NodesAtPosition NodesAtPosition
func (c *ChromeDpCV) NodesAtPosition(position *BrowserWindowPosition, resultNodes *[]*cdp.Node) chromedp.Action {
	return chromedp.ActionFunc(func(ctxt context.Context) error {
		var xPaths []string
		err := chromedp.Evaluate(javascript.GetElementsXPathForPoint(position.X, position.Y), &xPaths).Do(ctxt)
		if err != nil {
			return c.errorDebug(err)
		}

		for _, xpath := range xPaths {
			xpath = strings.ToLower("//" + xpath)

			var tmpNodes []*cdp.Node
			err = chromedp.Nodes(xpath, &tmpNodes, chromedp.BySearch).Do(ctxt)
			if err != nil {
				return c.errorDebug(err)
			}
			if len(tmpNodes) == 0 {
				return c.errorDebug(errors.New("no node found for xpath: " + xpath))
			}
			if len(tmpNodes) > 1 {
				return c.errorDebug(errors.New("more then one node found for xpath " + xpath))
			}
			*resultNodes = append(*resultNodes, tmpNodes...)
		}
		return nil
	})
}

// PositionWhereScreenLooksLike PositionWhereScreenLooksLike
func (c *ChromeDpCV) PositionWhereScreenLooksLike(targetImagePath string, position *BrowserWindowPosition) chromedp.Action {
	return chromedp.ActionFunc(func(ctxt context.Context) error {
		c.printDebug("Getting position for " + targetImagePath)
		if c == nil {
			return errors.New("c can't be nil for GetPositionOfRegionThatLooksLike func")
		}
		// 1. Create screenShot and save to temporary file
		screenShotFilePath, err := screenShotToTempFile(ctxt)
		if err != nil {
			return err
		}
		defer func() { logOnlyIfErr(os.Remove(screenShotFilePath)) }()

		exists := fileExists(screenShotFilePath)
		if !exists {
			return errors.Errorf("temp file %s doesn't exist\n", screenShotFilePath)
		}

		// 2. Read screenShot data
		screenShotMat := gocv.IMRead(screenShotFilePath, c.IMReadFlag)
		if screenShotMat.Empty() {
			return errors.Errorf("gocv: cannot read image %s\n", screenShotFilePath)
		}
		defer func() { logOnlyIfErr(screenShotMat.Close()) }()

		// 3. Read targetImage data
		targetImageMat := gocv.IMRead(targetImagePath, c.IMReadFlag)
		if targetImageMat.Empty() {
			return errors.Errorf("gocv: cannot read image %s\n", targetImagePath)
		}
		defer func() { logOnlyIfErr(targetImageMat.Close()) }()

		// 4. Find image-region left upper corner coordinates within screenShot
		resultImage := gocv.NewMatWithSize(screenShotMat.Rows(), screenShotMat.Cols(), 0)
		gocv.MatchTemplate(screenShotMat, targetImageMat, &resultImage, c.TemplateMatchMode, gocv.NewMat())
		_, _, minLoc, maxLoc := gocv.MinMaxLoc(resultImage)
		var matchLoc image.Point
		if c.TemplateMatchMode == gocv.TmSqdiff ||
			c.TemplateMatchMode == gocv.TmSqdiffNormed {
			matchLoc = minLoc
		} else {
			matchLoc = maxLoc
		}

		// Optional: save screenshots with marked targetImage region
		if c.TemplateMatchMarkedScreenShotFilePath != "" {
			gocv.Rectangle(
				&screenShotMat,
				image.Rect(
					matchLoc.X,
					matchLoc.Y,
					matchLoc.X+targetImageMat.Cols(),
					matchLoc.Y+targetImageMat.Rows()),
				color.RGBA{R: 239, A: 1, B: 26, G: 26},
				4)

			didWrite := gocv.IMWrite(c.TemplateMatchMarkedScreenShotFilePath, screenShotMat)
			if !didWrite {
				return errors.Errorf(
					"gocv: error writing marked screenshot to file: %s",
					c.TemplateMatchMarkedScreenShotFilePath)
			}
		}

		// 5. Calculate click coordinates
		// and be aware of differences between image size and BrowserWindow size (retina displays)
		screenShotWidth, screenShotHeight := float64(screenShotMat.Cols()), float64(screenShotMat.Rows())
		window := BrowserWindow{}
		err = chromedp.Evaluate(javascript.WindowSize(), &window).Do(ctxt)
		if err != nil {
			return errors.Wrap(err, `Evaluate("BrowserWindow.innerWidth;"`)
		}
		widthRatio, heightRatio := screenShotWidth/float64(window.Width), screenShotHeight/float64(window.Height)
		c.printDebug(fmt.Sprintf("w: %f -- h: %f", widthRatio, heightRatio))
		targetImageX0, targetImageY0 := float64(matchLoc.X), float64(matchLoc.Y)
		c.printDebug(fmt.Sprintf("w: %f -- h: %f", targetImageX0, targetImageY0))
		targetImageWidth, targetImageHeight := float64(targetImageMat.Cols()), float64(targetImageMat.Rows())
		c.printDebug(fmt.Sprintf("w: %f -- h: %f", targetImageWidth, targetImageHeight))

		if position == nil {
			position = &BrowserWindowPosition{}
		}
		position.window = &window
		position.X = (targetImageX0 / widthRatio) + (targetImageWidth / widthRatio / 2)
		position.Y = (targetImageY0 / heightRatio) + (targetImageHeight / heightRatio / 2)
		return nil
	})
}

func (c *ChromeDpCV) printDebug(str string) {
	if c.Debug {
		fmt.Println(str)
	}
}

func (c *ChromeDpCV) errorDebug(err error) error {
	if err != nil && c.Debug {
		log.Printf("%+v\n", err)
	}
	return err
}
