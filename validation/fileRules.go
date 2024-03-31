package validation

import (
	"github.com/gabriel-vasile/mimetype"
)

func CheckFileSize(fileSize, maxSize uint64) bool {
	return !(fileSize > (maxSize * 1024 * 1024))
}

func CheckFileMime(fileMime string) bool {
	allowed := []string{"image/webp", "image/jpg", "image/jpeg", "image/png"}

	return mimetype.EqualsAny(fileMime, allowed...)
}
