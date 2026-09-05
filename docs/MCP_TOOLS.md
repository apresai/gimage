# Gimage MCP Tools Reference

Complete reference for all 10 MCP tools available in the gimage server.

## Tool Index

1. [generate_image](#generate_image) - AI image generation
2. [resize_image](#resize_image) - Resize to dimensions
3. [scale_image](#scale_image) - Scale by factor
4. [crop_image](#crop_image) - Crop to region
5. [compress_image](#compress_image) - Compress file size
6. [convert_image](#convert_image) - Convert formats
7. [batch_resize](#batch_resize) - Batch resize
8. [batch_compress](#batch_compress) - Batch compress
9. [batch_convert](#batch_convert) - Batch convert
10. [list_models](#list_models) - List AI models

---

## generate_image

Generate an AI image from a text prompt using Gemini, Vertex AI, or xAI Grok.

### Description

Creates images from text descriptions using state-of-the-art AI models. Supports multiple models (Gemini 2.5 Flash, Gemini 3 Pro, Gemini 3.1 Flash via Vertex (vertex-flash), Grok Imagine/Quality), various sizes up to native 4K with Gemini 3+ models, and style controls. Options the chosen provider does not support are stripped and reported in the response `warning` field.

### Parameters

| Parameter | Type | Required | Default | Description |
| --------------- | ------- | -------- | ---------------------------- | ---------------------------------------------------------------------------------------------- |
| `prompt` | string | Yes | - | Text description of the image to generate |
| `output` | string | No | Auto-generated | Output file path |
| `size` | string | No | "1024x1024" | Image dimensions |
| `model` | string | No | "gemini-3-pro-image" | AI model to use |
| `image_size` | string | No | - | Native resolution. Gemini 3 Pro / 3.1 Flash: "1K", "2K", or "4K"; Flash Lite: "1K" only; Grok: "1K" or "2K" |
| `aspect_ratio` | string | No | - | Aspect ratio. Gemini 3+ includes 21:9 and ultra-wide. Grok Imagine also: "2:1", "1:2", "19.5:9", "9:19.5", "20:9", "9:20", "21:9", "5:2", "auto" |
| `style` | string | No | - | Image style (photorealistic, artistic, anime) |
| `seed` | integer | No | - | Random seed for reproducibility |
| `count` | integer | No | 1 | Number of images to generate (1-10; Grok returns N exactly, Gemini family is best-effort) |
| `output_format` | string | No | - | Output format for Vertex AI: "png", "jpeg", or "webp" |
| `thinking` | string | No | - | Reasoning depth for Gemini 3+ (`minimal`, `low`, `medium`, `high`). Ignored by Gemini 2.5 Flash and non-Gemini providers. |
| `grounding` | boolean | No | `false` | Enable Google Search grounding for Gemini 3+. Billed per search query in addition to per-image cost. |
| `quality` | string | No | - | Grok Imagine 2.0 only: `low`, `medium`, or `auto`. |
| `input_images` | array of strings | No | - | Reference images for editing/composition. Local paths (PNG/JPEG/WebP) for all providers; Grok also accepts public https:// URLs. Caps: Grok 2.0=5, older Grok=3, Gemini 2.5 Flash=3, Gemini 3 Pro=11, Gemini 3.1 Flash/Lite=14. Grok uses POST /v1/images/edits. |
| `user` | string | No | - | Optional end-user identifier for abuse monitoring (Grok/xAI only). |

### Supported Sizes

**Gemini & Vertex AI:**

- `512x512`
- `1024x1024` (default)
- `1024x1792`
- `1792x1024`
- native 1K/2K/4K (Gemini 3+ via `image_size` parameter)

### Supported Models

**Google Gemini API:**

- **gemini-3-pro-image** (default, native 4K, sharp text, $0.134 for 1K/2K, $0.24 for 4K)
- **gemini-3.1-flash-image** (native 4K at flash speed, tiered pricing: $0.045/0.5K, $0.067/1K, $0.101/2K, $0.151/4K)
- **gemini-3.1-flash-lite-image** (`flash` / `gemini-flash`, $0.034/image, 1K only)
- **gemini-2.5-flash-image** (legacy, $0.039/image, up to 1024x1024)

**Google Vertex AI:**

- **vertex-flash** (`vertex/flash-3.1`, medium thinking default)
- **vertex-flash-fast** (`vertex/flash-3.1-fast`, minimal thinking default)
- **vertex-flash-ultra** (`vertex/flash-3.1-ultra`, high thinking default)
- **vertex-flash-lite** (`vertex/flash-3.1-lite`, Flash Lite, $0.034/image, 1K only)
- `vertex-flash*` use `gemini-3.1-flash-image` via Vertex generateContent (Lite uses `gemini-3.1-flash-lite-image`)
- Pricing follows Gemini 3.1 Flash tiers except Lite ($0.034/1K)
- Options unsupported by the chosen provider are reported in the response `warning` field and ignored.

**xAI Grok:**

- **grok-imagine-image-2.0** (current Quality Mode, $0.04/image; `quality` low|medium|auto; default `grok` alias)
- **grok-imagine-image** (speed tier, $0.02/image; alias `grok-fast`)
- **grok-imagine-image-quality** (previous quality tier, $0.05 at 1K, $0.07 at 2K; xAI `-pro` aliases still resolve here)
 - Note: Grok models do not support size, style, or seed parameters (reported as warnings if passed)
 - Grok Imagine models support `aspect_ratio` (16 values including `auto`), `image_size` (1K or 2K), and `input_images` (max 5 on 2.0, max 3 on older models; routes to `/images/edits`)

### Returns

```json
{
 "success": true,
 "output_path": "/absolute/path/to/generated_1.png",
 "saved_paths": [
 "/absolute/path/to/generated_1.png",
 "/absolute/path/to/generated_2.png"
 ],
 "count": 2,
 "size": "1024x1024",
 "model": "gemini-3-pro-image",
 "prompt": "a sunset over mountains"
}
```

### Examples

**Basic generation (Gemini 3 Pro - default):**

```
Generate an image of a sunset over mountains
```

**With style:**

```
Create a photorealistic image of a wise old wizard
```

**Gemini 3 Pro with native 4K resolution:**

```
Generate a 4K image of a detailed infographic about space exploration
```

**Gemini 3 Pro with aspect ratio:**

```
Generate a 16:9 wide landscape image of mountains at sunset
```

**With size and in-prompt exclusions:**

```
Generate a 1024x1792 image of a forest scene with no people and no buildings
```

**Vertex AI with output format:**

```
Generate an image in WebP format of a modern architecture building
```


**Batch generation (multiple images):**

```
Generate 3 variations of a fantasy castle
```

**Reproducible generation:**

```
Generate an image with seed 42 of abstract patterns
```

**Using Gemini 2.5 Flash (affordable):**

```
Generate an image of a cat using gemini-2.5-flash-image model for affordable generation
```

**Gemini 3.1 Flash via Vertex (minimal thinking):**

```
Generate a quick product mockup using vertex-flash-fast model
```

**Gemini 3.1 Flash via Vertex (high thinking):**

```
Generate a premium architectural visualization using vertex-flash-ultra model
```

**xAI Grok Imagine 2.0 (current Quality Mode, $0.04/image):**

```
Generate a creative artistic image using grok model
```

**xAI Grok Imagine speed tier ($0.02/image):**

```
Generate a quick concept using grok-fast model
```

**xAI Grok with aspect ratio:**

```
Generate a 16:9 landscape using grok model with wide aspect ratio
```

---

## resize_image

Resize an image to specific dimensions.

### Description

Resizes an image to target width and height using high-quality Lanczos resampling. Supports three modes:
- **crop** (default): Preserves aspect ratio by filling the target size and cropping excess from center. Best for thumbnails, avatars, and social media images.
- **fit**: Preserves aspect ratio by fitting within target bounds. Result may be smaller than target dimensions. Best for constrained display areas.
- **stretch**: Forces exact dimensions regardless of aspect ratio. May distort the image. Best for exact sizing requirements.

### Parameters

| Parameter | Type | Required | Default | Description |
| --------- | ------- | -------- | ------- | ------------------------------------------------------------------------------------------------------------------------------------------- |
| `input` | string | Yes | - | Input image file path |
| `width` | integer | Yes | - | Target width in pixels (minimum: 1) |
| `height` | integer | Yes | - | Target height in pixels (minimum: 1) |
| `mode` | string | No | "crop" | Resize mode: "crop" (fill & crop), "fit" (fit within bounds), or "stretch" (force exact dimensions) |
| `output` | string | No | auto | Output file path (default: auto-generated) |

### Resize Modes Explained

| Mode | Aspect Ratio | Output Size | Use Case |
| ------- | ------------ | ------------------------ | ------------------------------ |
| crop | Preserved | Exact target dimensions | Thumbnails, avatars, cards |
| fit | Preserved | Fits within target | Galleries, constrained layouts |
| stretch | Not preserved| Exact target dimensions | Exact sizing, backgrounds |

**Example with 200x100 image (2:1 ratio) resized to 100x100 target:**
- `crop`: 100x100 (scales height to 100, crops width to fit)
- `fit`: 100x50 (fits 200x100 within 100x100, maintaining 2:1 ratio)
- `stretch`: 100x100 (distorts from 2:1 to 1:1 ratio)

### Returns

```json
{
 "success": true,
 "output_path": "/absolute/path/to/photo_resized.jpg",
 "original_size": "3000x2000",
 "new_size": "800x600",
 "mode": "crop"
}
```

### Examples

```
Resize photo.jpg to 800x600 pixels (uses crop mode by default)
Resize landscape.png to 1920x1080 using fit mode
Resize avatar.jpg to 100x100 with stretch mode for exact dimensions
```

---

## scale_image

Scale an image by a factor while preserving aspect ratio.

### Description

Scales an image proportionally by a multiplication factor. Use this when you want to make an image larger or smaller while maintaining its aspect ratio. Uses high-quality Lanczos resampling.

### Parameters

| Parameter | Type | Required | Description |
| --------- | ------ | -------- | ------------------------------------------ |
| `input` | string | Yes | Input image file path |
| `factor` | number | Yes | Scale factor (0.1 to 10.0) |
| `output` | string | No | Output file path (default: auto-generated) |

### Scale Factor Examples

- `0.5` = Half size
- `0.25` = Quarter size
- `2.0` = Double size
- `1.5` = 50% larger

### Returns

```json
{
 "success": true,
 "output_path": "/absolute/path/to/photo_scaled.jpg",
 "scale_factor": 0.5,
 "original_size": "2000x1500",
 "new_size": "1000x750"
}
```

### Examples

```
Scale photo.jpg to half its size
Make image.png twice as large (factor 2.0)
Reduce dimensions by 25% (factor 0.75)
```

---

## crop_image

Crop an image to a specific rectangular region.

### Description

Extracts a rectangular region from an image. Specify the top-left corner coordinates (x, y) and the width and height of the region. Useful for removing borders, focusing on specific areas, or extracting thumbnails.

### Parameters

| Parameter | Type | Required | Description |
| --------- | ------- | -------- | ----------------------------------------------- |
| `input` | string | Yes | Input image file path |
| `x` | integer | Yes | X coordinate of top-left corner (0 = left edge) |
| `y` | integer | Yes | Y coordinate of top-left corner (0 = top edge) |
| `width` | integer | Yes | Width of crop region in pixels (minimum: 1) |
| `height` | integer | Yes | Height of crop region in pixels (minimum: 1) |
| `output` | string | No | Output file path (default: auto-generated) |

### Returns

```json
{
 "success": true,
 "output_path": "/absolute/path/to/photo_cropped.jpg",
 "crop_region": "(100,100,800,600)",
 "crop_size": "800x600"
}
```

### Examples

```
Crop photo.jpg starting at (100, 100) with width 800 and height 600
Extract a 500x500 square from the top-left corner of image.png
```

---

## compress_image

Compress an image to reduce file size.

### Description

Reduces image file size while maintaining visual quality. Quality ranges from 1 (lowest quality, smallest file) to 100 (highest quality, largest file). Default is 90 which provides excellent quality with good compression. Most effective on JPEG images. PNG images are compressed losslessly.

### Parameters

| Parameter | Type | Required | Default | Description |
| --------- | ------- | -------- | -------------- | --------------------------- |
| `input` | string | Yes | - | Input image file path |
| `quality` | integer | No | 90 | Compression quality (1-100) |
| `output` | string | No | Auto-generated | Output file path |

### Recommended Quality Settings

- **95-100**: Archival quality, minimal compression
- **90**: Recommended for web (default)
- **85**: Good for mobile devices
- **75**: Acceptable for thumbnails
- **60-70**: Heavy compression, visible quality loss

### Returns

```json
{
 "success": true,
 "output_path": "/absolute/path/to/photo_compressed.jpg",
 "quality": 85,
 "original_size_bytes": 2500000,
 "compressed_size_bytes": 450000,
 "compression_ratio": "0.18",
 "savings_bytes": 2050000,
 "savings_percent": "82.0%",
 "original_size_human": "2.4 MB",
 "compressed_size_human": "439.5 KB"
}
```

### Examples

```
Compress photo.jpg to 85% quality
Reduce file size of large-image.png
Compress with 75% quality for thumbnails
```

---

## convert_image

Convert an image between different formats.

### Description

Converts images between PNG, JPG/JPEG, WebP, GIF, TIFF, and BMP formats. Useful for web optimization (converting to WebP), compatibility (PNG to JPG), or specific application requirements. Format detection is automatic based on file extension.

### Parameters

| Parameter | Type | Required | Description |
| --------- | ------ | -------- | ------------------------------------------------------------- |
| `input` | string | Yes | Input image file path |
| `format` | string | Yes | Target format (png, jpg, jpeg, webp, gif, tiff, bmp) |
| `output` | string | No | Output file path (default: auto-generated with new extension) |

### Supported Formats

- **PNG**: Lossless, supports transparency
- **JPG/JPEG**: Lossy, best for photos
- **WebP**: Modern format, great compression
- **GIF**: Animated images, limited colors
- **TIFF**: High-quality, large files
- **BMP**: Uncompressed, very large files

### Returns

```json
{
 "success": true,
 "output_path": "/absolute/path/to/image.webp",
 "original_format": "png",
 "new_format": "webp",
 "original_size": "1.2 MB",
 "new_size": "245.3 KB"
}
```

### Examples

```
Convert photo.png to JPEG format
Change image.jpg to WebP for better web performance
Convert screenshot.bmp to PNG
```

---

## batch_resize

Resize multiple images concurrently.

### Description

Processes all image files (PNG, JPG, WebP, GIF, TIFF, BMP) in a directory and resizes them to specified dimensions. Uses parallel workers for fast processing of large batches. Supports the same three resize modes as `resize_image`: crop (default), fit, and stretch.

### Parameters

| Parameter | Type | Required | Default | Description |
| ------------ | ------- | -------- | --------- | ----------------------------------------------------------------------- |
| `input_dir` | string | Yes | - | Input directory containing images |
| `width` | integer | Yes | - | Target width in pixels (minimum: 1) |
| `height` | integer | Yes | - | Target height in pixels (minimum: 1) |
| `mode` | string | No | "crop" | Resize mode: "crop" (fill & crop), "fit" (fit within bounds), "stretch" |
| `output_dir` | string | Yes | - | Output directory (created if doesn't exist) |
| `workers` | integer | No | CPU cores | Number of parallel workers (1-16) |

### Returns

```json
{
 "success": true,
 "processed": 45,
 "failed": 2,
 "total": 47,
 "output_dir": "/absolute/path/to/output",
 "errors": [
 "corrupted.jpg: failed to decode image",
 "locked.png: permission denied"
 ]
}
```

### Examples

```
Resize all images in vacation-photos folder to 1920x1080, save to resized-photos
Batch resize images in products/ to 600x600 using crop mode (default)
Batch resize images in gallery/ to 800x600 using fit mode to preserve aspect ratios
```

---

## batch_compress

Compress multiple images concurrently.

### Description

Processes all image files in a directory with specified quality setting to reduce file sizes. Reports total space saved across all images. Uses parallel workers for efficient processing.

### Parameters

| Parameter | Type | Required | Default | Description |
| ------------ | ------- | -------- | --------- | ------------------------------------------- |
| `input_dir` | string | Yes | - | Input directory containing images |
| `quality` | integer | No | 85 | Compression quality (1-100) |
| `output_dir` | string | Yes | - | Output directory (created if doesn't exist) |
| `workers` | integer | No | CPU cores | Number of parallel workers (1-16) |

### Returns

```json
{
 "success": true,
 "processed": 50,
 "failed": 0,
 "total": 50,
 "output_dir": "/absolute/path/to/compressed",
 "total_original_size": "125.5 MB",
 "total_new_size": "28.3 MB",
 "total_savings": "97.2 MB",
 "savings_percent": "77.5%"
}
```

### Examples

```
Compress all images in photos/ to 85% quality, save to compressed/
Batch compress with 90% quality using 8 workers
```

---

## batch_convert

Convert multiple images to a different format concurrently.

### Description

Converts all image files in a directory to a specified format. Useful for converting entire directories to WebP for web optimization, or to PNG for lossless archival. Maintains original filenames with new extensions.

### Parameters

| Parameter | Type | Required | Default | Description |
| ------------ | ------- | -------- | --------- | ---------------------------------------------------- |
| `input_dir` | string | Yes | - | Input directory containing images |
| `format` | string | Yes | - | Target format (png, jpg, jpeg, webp, gif, tiff, bmp) |
| `output_dir` | string | Yes | - | Output directory (created if doesn't exist) |
| `workers` | integer | No | CPU cores | Number of parallel workers (1-16) |

### Returns

```json
{
 "success": true,
 "processed": 30,
 "failed": 0,
 "total": 30,
 "output_dir": "/absolute/path/to/webp-images"
}
```

### Examples

```
Convert all images in photos/ to WebP format, save to webp-images/
Batch convert to PNG using 8 workers for faster processing
```

---

## list_models

List all available AI image generation models.

### Description

Returns detailed information about all available AI image generation models, including their capabilities, providers, maximum resolutions, and authentication requirements. Use this to discover which models are available before generating images.

### Parameters

None

### Returns

```json
{
 "providers": [
 {
 "provider_id": "gemini/pro-3",
 "name": "Gemini 3 Pro (via Gemini API)",
 "api": "gemini",
 "model_id": "gemini-3-pro-image",
 "available": true,
 "pricing_summary": "$0.1340/image",
 "supports_styles": true,
 "supports_seed": true,
 "supports_image_size": true,
 "supports_aspect_ratio": true,
 "supports_thinking": true,
 "supports_grounding": true,
 "supports_input_images": true,
 "supports_output_format": false,
 "supports_multiple_images": true
 }
 // ... more providers
 ],
 "total": 9,
 "configured": 1
}
```

### Examples

```
List all available AI models
Show me what image generation models I can use
What models support 2K resolution?
```

---

## Error Handling

All tools return errors in a consistent format:

```json
{
 "error": {
 "code": -32603,
 "message": "Tool execution failed: file not found: /path/to/missing.jpg"
 }
}
```

### Common Error Codes

- **-32602**: Invalid parameters (missing required field, invalid type, out of range)
- **-32603**: Execution error (file not found, permission denied, API error)
- **-32601**: Method not found (invalid tool name)

### Error Messages

Error messages are designed to be clear and actionable:

- ✓ "Failed to open image: file not found: /path/to/photo.jpg"
- ✓ "Crop region (100,100,2000,1500) extends beyond image bounds (1000x800)"
- ✓ "Quality must be between 1 and 100, got: 150"

---

## Performance Notes

### Single Image Operations

- **Fast**: resize, scale, crop, convert (< 1 second for typical images)
- **Medium**: compress (1-3 seconds depending on size and quality)
- **Slow**: generate (5-30 seconds depending on model and size)
 - Gemini 2.5 Flash: ~5-10 seconds
 - Gemini 3.1 Flash via Vertex (vertex-flash): ~10-20 seconds

### Batch Operations

- Uses parallel workers (default: number of CPU cores)
- Processing 100 images:
 - Resize: ~10-30 seconds (4 workers)
 - Compress: ~20-60 seconds (4 workers)
 - Convert: ~15-45 seconds (4 workers)

### Provider-Specific Parameters

Different providers support different advanced parameters:

**Gemini (Flash & Pro) and Vertex AI:**

- `aspect_ratio`: Control aspect ratio (1:1, 16:9, 9:16, 4:3, 3:4, 3:2, 2:3)

**Gemini 3+ and Grok Imagine:**

- `image_size`: Native resolution upscaling. Gemini 3 Pro / 3.1 Flash accept `1K`, `2K`, `4K`. Grok Imagine and Grok Imagine Quality accept `1K`, `2K` only (4K emits a stderr warning and falls back to xAI default).

**Vertex AI Only:**

- `output_format`: Control output format (png, jpeg, webp)

**Grok Imagine Only:**

- `aspect_ratio`: Control aspect ratio (1:1, 16:9, 9:16, 4:3, 3:4, etc.)

**Gemini and Vertex AI (not Grok):**

- `seed`: Random seed for reproducibility (not supported by Grok; reported as a warning if passed)

**All Providers:**

- `count`: Generate multiple images (max varies: Gemini 4, Vertex 8, Grok 10)

### Tips for Better Performance

1. Use batch operations for multiple images instead of repeated single operations
2. Increase workers for faster batch processing (up to 16)
3. Use smaller images when possible
4. For generation:
 - Use **Gemini 3.1 Flash Lite** for cheapest Gemini generation ($0.034/image, 1K only)
 - Use **Gemini 3 Pro** for highest quality text/diagrams ($0.134-$0.24/image)
 - Use **vertex-flash-lite** for Gemini 3.1 Flash Lite via Vertex ($0.034/image, minimal thinking default)
 - Use **vertex-flash** for Gemini 3.1 Flash via Vertex ($0.045-$0.151/image, medium thinking default)
 - Use **vertex-flash-fast** for the same Vertex backend with minimal thinking default
 - Use **vertex-flash-ultra** for the same Vertex backend with high thinking default
 - Use **Grok Imagine 2.0** (`grok` / `grok-quality`) for current xAI Quality Mode ($0.04/image)
 - Use **Grok Imagine** (`grok-fast`) for the $0.02/image speed tier
 - Use **Grok Imagine Quality** (`grok-imagine-image-quality`) for the older quality tier ($0.05 at 1K, $0.07 at 2K)
5. Use `count` parameter for batch generation instead of calling generate_image multiple times

---

## See Also

- [Usage Guide](MCP_USAGE.md) - Complete setup and usage instructions
- [Examples](MCP_EXAMPLES.md) - Real-world usage examples
- [Main Documentation](../README.md) - Full project documentation
