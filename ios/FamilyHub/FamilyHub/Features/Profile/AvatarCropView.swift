import SwiftUI
import UIKit

struct AvatarCropView: View {
    let image: UIImage
    let onConfirm: (UIImage) -> Void

    @Environment(\.dismiss) private var dismiss
    @State private var zoom: CGFloat = 1
    @State private var committedZoom: CGFloat = 1
    @State private var offset = CGSize.zero
    @State private var committedOffset = CGSize.zero
    @State private var errorMessage: String?
    @State private var viewportSide: CGFloat = 0

    var body: some View {
        NavigationStack {
            VStack(spacing: 24) {
                GeometryReader { geometry in
                    let side = min(geometry.size.width, geometry.size.height)
                    let baseScale = max(side / image.size.width, side / image.size.height)
                    let scale = baseScale * zoom

                    ZStack {
                        AppTheme.surface
                        Image(uiImage: image)
                            .resizable()
                            .frame(width: image.size.width * scale, height: image.size.height * scale)
                            .position(x: side / 2 + offset.width, y: side / 2 + offset.height)
                            .gesture(dragGesture(side: side, baseScale: baseScale))
                            .simultaneousGesture(zoomGesture(side: side, baseScale: baseScale))

                        CropOutsideOverlay()
                            .fill(.black.opacity(0.55), style: FillStyle(eoFill: true))
                            .allowsHitTesting(false)

                        Circle()
                            .strokeBorder(.white, lineWidth: 2)
                            .shadow(color: .black.opacity(0.35), radius: 3)
                            .allowsHitTesting(false)
                    }
                    .frame(width: side, height: side)
                    .clipShape(Rectangle())
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
                    .accessibilityLabel("Photo crop preview")
                    .onAppear { viewportSide = side }
                    .onChange(of: side) { _, newSide in viewportSide = newSide }
                }
                .aspectRatio(1, contentMode: .fit)
                .padding(.horizontal, 24)

                Text("Drag to position. Pinch to zoom.")
                    .font(.subheadline)
                    .foregroundStyle(AppTheme.secondary)
            }
            .padding(.vertical, 24)
            .navigationTitle("Crop Photo")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") { dismiss() }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Use Photo") { confirmCrop() }
                }
            }
            .alert("Couldn’t prepare photo", isPresented: Binding(
                get: { errorMessage != nil },
                set: { if !$0 { errorMessage = nil } }
            )) {
                Button("OK", role: .cancel) {}
            } message: {
                Text(errorMessage ?? "Try choosing another image.")
            }
        }
        .presentationDetents([.large])
    }

    private func dragGesture(side: CGFloat, baseScale: CGFloat) -> some Gesture {
        DragGesture()
            .onChanged { value in
                offset = boundedOffset(
                    CGSize(
                        width: committedOffset.width + value.translation.width,
                        height: committedOffset.height + value.translation.height
                    ),
                    side: side,
                    scale: baseScale * zoom
                )
            }
            .onEnded { value in
                committedOffset = boundedOffset(
                    CGSize(
                        width: committedOffset.width + value.translation.width,
                        height: committedOffset.height + value.translation.height
                    ),
                    side: side,
                    scale: baseScale * zoom
                )
                offset = committedOffset
            }
    }

    private func zoomGesture(side: CGFloat, baseScale: CGFloat) -> some Gesture {
        MagnificationGesture()
            .onChanged { value in
                zoom = min(max(committedZoom * value, 1), 5)
                offset = boundedOffset(offset, side: side, scale: baseScale * zoom)
            }
            .onEnded { value in
                committedZoom = min(max(committedZoom * value, 1), 5)
                zoom = committedZoom
                committedOffset = boundedOffset(offset, side: side, scale: baseScale * zoom)
                offset = committedOffset
            }
    }

    private func boundedOffset(_ proposed: CGSize, side: CGFloat, scale: CGFloat) -> CGSize {
        let horizontalLimit = max(0, (image.size.width * scale - side) / 2)
        let verticalLimit = max(0, (image.size.height * scale - side) / 2)
        return CGSize(
            width: min(max(proposed.width, -horizontalLimit), horizontalLimit),
            height: min(max(proposed.height, -verticalLimit), verticalLimit)
        )
    }

    private func confirmCrop() {
        let side = min(viewportSide, 640)
        guard side > 0 else {
            errorMessage = ProfilePhotoError.encodingFailed.localizedDescription
            return
        }
        let baseScale = max(side / image.size.width, side / image.size.height)
        let scale = baseScale * zoom
        let cropSide = side / scale
        let originX = (image.size.width - cropSide) / 2 - offset.width / scale
        let originY = (image.size.height - cropSide) / 2 - offset.height / scale
        let outputSize = CGSize(width: 512, height: 512)
        let format = UIGraphicsImageRendererFormat()
        format.scale = 1
        format.opaque = true
        let renderer = UIGraphicsImageRenderer(size: outputSize, format: format)
        let croppedImage = renderer.image { _ in
            UIColor.white.setFill()
            UIRectFill(CGRect(origin: .zero, size: outputSize))
            image.draw(in: CGRect(
                x: -originX * outputSize.width / cropSide,
                y: -originY * outputSize.height / cropSide,
                width: image.size.width * outputSize.width / cropSide,
                height: image.size.height * outputSize.height / cropSide
            ))
        }
        onConfirm(croppedImage)
        dismiss()
    }
}

private struct CropOutsideOverlay: Shape {
    func path(in rect: CGRect) -> Path {
        var path = Path()
        path.addRect(rect)
        path.addEllipse(in: rect)
        return path
    }
}

enum AvatarPhotoProcessor {
    static func jpegData(for image: UIImage) -> Data? {
        var size = image.size
        var resizedImage = image

        while true {
            for quality in [0.82, 0.68, 0.54, 0.4] {
                guard let data = resizedImage.jpegData(compressionQuality: quality) else { return nil }
                if data.count <= 1_800_000 { return data }
            }

            size = CGSize(width: size.width * 0.75, height: size.height * 0.75)
            guard size.width >= 64, size.height >= 64,
                  let downsampledImage = Self.downsample(resizedImage, to: size) else { return nil }
            resizedImage = downsampledImage
        }
    }

    private static func downsample(_ image: UIImage, to size: CGSize) -> UIImage? {
        let format = UIGraphicsImageRendererFormat()
        format.scale = 1
        format.opaque = true
        let renderer = UIGraphicsImageRenderer(size: size, format: format)
        return renderer.image { _ in
            UIColor.white.setFill()
            UIRectFill(CGRect(origin: .zero, size: size))
            image.draw(in: CGRect(origin: .zero, size: size))
        }
    }
}

enum ProfilePhotoError: LocalizedError {
    case invalidImage
    case encodingFailed

    var errorDescription: String? {
        switch self {
        case .invalidImage: "This image can’t be opened. Choose another photo."
        case .encodingFailed: "This photo couldn’t be prepared. Choose another image."
        }
    }
}
