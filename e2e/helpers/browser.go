// Package helpers provides narrowly-scoped utilities for E2E testing.
package helpers

import (
	"fmt"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

// Browser provides headless browser capabilities for E2E tests.
//
// Use this helper to:
//   - Test UI flows and user interactions
//   - Verify page content and element states
//   - Test HTMX-powered dynamic updates
//
// Browser intentionally exposes only five methods (Navigate, Click, Fill, Text, Wait)
// to keep the E2E testing interface minimal and focused.
type Browser struct {
	browser *rod.Browser
	page    *rod.Page
	timeout time.Duration
}

// NewBrowser creates a new headless browser instance.
//
// The browser runs in headless mode by default. Call Close() when done
// to release browser resources.
//
//	browser, err := NewBrowser()
//	if err != nil {
//	    t.Fatal(err)
//	}
//	defer browser.Close()
func NewBrowser() (*Browser, error) {
	url := launcher.New().Headless(true).MustLaunch()

	browser := rod.New().ControlURL(url)
	if err := browser.Connect(); err != nil {
		return nil, fmt.Errorf("failed to connect to browser: %w", err)
	}

	page := browser.MustPage("")

	return &Browser{
		browser: browser,
		page:    page,
		timeout: 30 * time.Second,
	}, nil
}

// SetTimeout sets the timeout for browser operations.
//
// The default timeout is 30 seconds. Use a shorter timeout for faster
// failure detection in tests.
func (b *Browser) SetTimeout(d time.Duration) {
	b.timeout = d
}

// Navigate loads the specified URL in the browser.
//
// Waits for the page to finish loading before returning.
//
//	err := browser.Navigate("http://localhost:8080/login")
func (b *Browser) Navigate(url string) error {
	if err := b.page.Timeout(b.timeout).Navigate(url); err != nil {
		return fmt.Errorf("failed to navigate to %s: %w", url, err)
	}
	if err := b.page.Timeout(b.timeout).WaitLoad(); err != nil {
		return fmt.Errorf("failed to wait for page load: %w", err)
	}
	return nil
}

// Click clicks on the element matching the CSS selector.
//
// Waits for the element to be visible before clicking.
//
//	err := browser.Click("button[type='submit']")
//	err := browser.Click("#login-button")
//	err := browser.Click(".nav-link.active")
func (b *Browser) Click(selector string) error {
	el, err := b.page.Timeout(b.timeout).Element(selector)
	if err != nil {
		return fmt.Errorf("failed to find element %s: %w", selector, err)
	}
	if err := el.Click(proto.InputMouseButtonLeft, 1); err != nil {
		return fmt.Errorf("failed to click element %s: %w", selector, err)
	}
	return nil
}

// Fill types text into the input element matching the CSS selector.
//
// Clears any existing value before typing the new value.
//
//	err := browser.Fill("input[name='email']", "test@example.com")
//	err := browser.Fill("#password", "secret123")
func (b *Browser) Fill(selector, value string) error {
	el, err := b.page.Timeout(b.timeout).Element(selector)
	if err != nil {
		return fmt.Errorf("failed to find element %s: %w", selector, err)
	}
	if err := el.SelectAllText(); err != nil {
		return fmt.Errorf("failed to select text in %s: %w", selector, err)
	}
	if err := el.Input(value); err != nil {
		return fmt.Errorf("failed to input text into %s: %w", selector, err)
	}
	return nil
}

// Text returns the text content of the element matching the CSS selector.
//
// Use this to verify page content in assertions.
//
//	text, err := browser.Text("h1.page-title")
//	assert.Equal(t, "Welcome", text)
func (b *Browser) Text(selector string) (string, error) {
	el, err := b.page.Timeout(b.timeout).Element(selector)
	if err != nil {
		return "", fmt.Errorf("failed to find element %s: %w", selector, err)
	}
	text, err := el.Text()
	if err != nil {
		return "", fmt.Errorf("failed to get text from %s: %w", selector, err)
	}
	return text, nil
}

// Wait waits for an element matching the CSS selector to appear.
//
// Use this to wait for HTMX updates or dynamic content.
//
//	err := browser.Wait(".success-message")
//	err := browser.Wait("[data-loaded='true']")
func (b *Browser) Wait(selector string) error {
	_, err := b.page.Timeout(b.timeout).Element(selector)
	if err != nil {
		return fmt.Errorf("timeout waiting for element %s: %w", selector, err)
	}
	return nil
}

// Close releases browser resources.
//
// Always call Close() when done with the browser, typically using defer:
//
//	browser, err := NewBrowser()
//	if err != nil {
//	    t.Fatal(err)
//	}
//	defer browser.Close()
func (b *Browser) Close() error {
	if b.page != nil {
		b.page.Close()
	}
	if b.browser != nil {
		return b.browser.Close()
	}
	return nil
}
