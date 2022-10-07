package render

import (
	"github.com/vulkan-go/vulkan"
)

type Context interface {
	SetOnPrepare(onPrepare func() error)
	SetOnCleanup(onCleanup func() error)
	SetOnInvalidate(onInvalidate func(imageIndex int) error)
	Device() vulkan.Device
	CommandBuffer() vulkan.CommandBuffer
	Platform() Platform
	SwapchainDimensions() *SwapchainDimensions
	SwapchainImageDimensions() []*SwapchainImageDimensions
	AcquireNextImage() (imageIndex int, outdated bool, err error)
	PresentImage(imageIndex int) (outdated bool, err error)
}

type context struct {}