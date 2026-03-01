// Copyright (c) EnvSync Contributors. SPDX-License-Identifier: MIT

package ui

import "fmt"

// Banner prints the EnvSync ASCII art banner.
func Banner(version string) string {
	brand := StyleBrand.Render

	art := brand("  ╔═══════════════════════════════╗") + "\n" +
		brand("  ║") + "  " + StyleBold.Render("EnvSync") + "  " + StyleDim.Render(version) + brand("          ║") + "\n" +
		brand("  ║") + "  " + StyleDim.Render("Secure .env sync for teams") + brand("  ║") + "\n" +
		brand("  ╚═══════════════════════════════╝")

	return art
}

// Header prints a formatted section header.
func Header(title string) {
	fmt.Println()
	fmt.Println(Indent(BrandIcon() + " " + StyleBold.Render(title)))
	fmt.Println()
}

// Status prints a status line with an icon.
func Status(msg string) {
	fmt.Println(Indent(InfoIcon() + " " + msg))
}

// Success prints a success line.
func Success(msg string) {
	fmt.Println(Indent(SuccessIcon() + " " + msg))
}

// Warning prints a warning line.
func Warning(msg string) {
	fmt.Println(Indent(WarningIcon() + " " + StyleWarning.Render(msg)))
}

// Error prints an error line.
func Error(msg string) {
	fmt.Println(Indent(ErrorIcon() + " " + StyleError.Render(msg)))
}

// Blank prints a blank line.
func Blank() {
	fmt.Println()
}

// Line prints the given text indented.
func Line(text string) {
	fmt.Println(Indent(text))
}

// Code prints text styled as code.
func Code(text string) {
	fmt.Println(Indent(StyleCode.Render(text)))
}
