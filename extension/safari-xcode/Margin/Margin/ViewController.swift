//
//  ViewController.swift
//  Margin
//
//  Created by Padding Labs LLC on 5/25/26.
//

import Cocoa
import SafariServices

let extensionBundleIdentifier = "at.margin.extension.Extension"

class ViewController: NSViewController {

    private let statusLabel = NSTextField(labelWithString: "Extension not yet enabled")
    private let instructionsLabel = NSTextField(labelWithString: "")
    private let readyLabel = NSTextField(labelWithString: "Ready to use on every website")
    private let openButton = NSButton(title: "Open Safari Settings", target: nil, action: nil)

    override func viewDidLoad() {
        super.viewDidLoad()
        buildUI()
    }

    override func viewWillAppear() {
        super.viewWillAppear()
        view.window?.title = "Margin"
        view.window?.setContentSize(NSSize(width: 440, height: 400))
        view.window?.center()
        refreshState()
    }

    // MARK: - UI

    private func buildUI() {
        let icon = NSImageView()
        icon.image = NSApplication.shared.applicationIconImage
        icon.imageScaling = .scaleProportionallyUpOrDown
        icon.translatesAutoresizingMaskIntoConstraints = false
        NSLayoutConstraint.activate([
            icon.widthAnchor.constraint(equalToConstant: 72),
            icon.heightAnchor.constraint(equalToConstant: 72),
        ])

        let title = NSTextField(labelWithString: "Margin")
        title.font = .systemFont(ofSize: 22, weight: .bold)
        title.alignment = .center

        let subtitle = NSTextField(wrappingLabelWithString:
            "Annotate and highlight any webpage, with your notes saved to the AT Protocol.")
        subtitle.alignment = .center
        subtitle.textColor = .secondaryLabelColor
        subtitle.font = .systemFont(ofSize: 13)
        subtitle.translatesAutoresizingMaskIntoConstraints = false
        subtitle.widthAnchor.constraint(lessThanOrEqualToConstant: 320).isActive = true

        statusLabel.alignment = .center
        statusLabel.textColor = .secondaryLabelColor
        statusLabel.font = .systemFont(ofSize: 12, weight: .medium)

        instructionsLabel.attributedStringValue = makeInstructions()
        instructionsLabel.isSelectable = false
        instructionsLabel.alignment = .left
        (instructionsLabel.cell as? NSTextFieldCell)?.wraps = true

        readyLabel.alignment = .center
        readyLabel.font = .systemFont(ofSize: 13, weight: .semibold)
        readyLabel.textColor = NSColor.systemGreen
        readyLabel.isHidden = true

        openButton.bezelStyle = .rounded
        openButton.controlSize = .large
        openButton.target = self
        openButton.action = #selector(openPreferences)
        openButton.keyEquivalent = "\r"

        let stack = NSStackView(views: [
            icon, title, subtitle, statusLabel, instructionsLabel, readyLabel, openButton,
        ])
        stack.orientation = .vertical
        stack.alignment = .centerX
        stack.spacing = 12
        stack.setCustomSpacing(6, after: icon)
        stack.setCustomSpacing(4, after: title)
        stack.setCustomSpacing(16, after: subtitle)
        stack.setCustomSpacing(16, after: statusLabel)
        stack.translatesAutoresizingMaskIntoConstraints = false

        view.addSubview(stack)
        NSLayoutConstraint.activate([
            stack.centerXAnchor.constraint(equalTo: view.centerXAnchor),
            stack.centerYAnchor.constraint(equalTo: view.centerYAnchor),
            stack.leadingAnchor.constraint(greaterThanOrEqualTo: view.leadingAnchor, constant: 32),
            stack.trailingAnchor.constraint(lessThanOrEqualTo: view.trailingAnchor, constant: -32),
        ])
    }

    private func makeInstructions() -> NSAttributedString {
        let body = NSFont.systemFont(ofSize: 12)
        let bold = NSFont.systemFont(ofSize: 12, weight: .semibold)
        let para = NSMutableParagraphStyle()
        para.lineSpacing = 5
        para.paragraphSpacing = 4

        let result = NSMutableAttributedString()
        func line(_ num: String, _ parts: [(String, Bool)]) {
            result.append(NSAttributedString(string: num + "  ", attributes: [
                .font: bold, .foregroundColor: NSColor.controlAccentColor,
            ]))
            for (text, isBold) in parts {
                result.append(NSAttributedString(string: text, attributes: [
                    .font: isBold ? bold : body,
                    .foregroundColor: isBold ? NSColor.labelColor : NSColor.secondaryLabelColor,
                ]))
            }
            result.append(NSAttributedString(string: "\n"))
        }

        line("1.", [("Open ", false), ("Safari ▸ Settings ▸ Extensions", true)])
        line("2.", [("Enable ", false), ("Margin", true), (" in the list", false)])
        line("3.", [("Set ", false), ("“Allow on”", false), (" to ", false), ("All Websites", true)])

        result.addAttribute(.paragraphStyle, value: para,
                            range: NSRange(location: 0, length: result.length))
        // drop the trailing newline
        if result.length > 0 { result.deleteCharacters(in: NSRange(location: result.length - 1, length: 1)) }
        return result
    }

    // MARK: - Extension state

    private func refreshState() {
        SFSafariExtensionManager.getStateOfSafariExtension(withIdentifier: extensionBundleIdentifier) { state, error in
            DispatchQueue.main.async {
                if let state = state, error == nil {
                    self.apply(enabled: state.isEnabled)
                } else {
                    self.apply(enabled: nil)
                }
            }
        }
    }

    private func apply(enabled: Bool?) {
        switch enabled {
        case .some(true):
            statusLabel.stringValue = "Extension is enabled"
            instructionsLabel.isHidden = true
            readyLabel.isHidden = false
        case .some(false):
            statusLabel.stringValue = "Extension is disabled"
            instructionsLabel.isHidden = false
            readyLabel.isHidden = true
        case .none:
            statusLabel.stringValue = "Extension not yet enabled"
            instructionsLabel.isHidden = false
            readyLabel.isHidden = true
        }
    }

    @objc private func openPreferences() {
        SFSafariApplication.showPreferencesForExtension(withIdentifier: extensionBundleIdentifier) { _ in
            DispatchQueue.main.async {
                NSApplication.shared.terminate(nil)
            }
        }
    }
}
