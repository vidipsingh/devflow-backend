package database

import (
	"bytes"
	"context"
	"io"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// GridFSUpload stores content bytes under filename and returns the new file's ObjectID.
// Uses mongo-driver v2: GridFSBucket() is a method on *mongo.Database with no arguments.
func GridFSUpload(ctx context.Context, filename string, content []byte) (bson.ObjectID, error) {
	bucket := database.GridFSBucket()

	uploadStream, err := bucket.OpenUploadStream(ctx, filename)
	if err != nil {
		return bson.ObjectID{}, err
	}
	defer uploadStream.Close()

	if _, err := io.Copy(uploadStream, bytes.NewReader(content)); err != nil {
		return bson.ObjectID{}, err
	}

	// In v2, FileID is a field of type any; assert to bson.ObjectID
	id, ok := uploadStream.FileID.(bson.ObjectID)
	if !ok {
		return bson.ObjectID{}, nil
	}
	return id, nil
}

// GridFSDownload retrieves the content for a given GridFS file ID.
func GridFSDownload(ctx context.Context, fileID bson.ObjectID) ([]byte, error) {
	bucket := database.GridFSBucket()

	downloadStream, err := bucket.OpenDownloadStream(ctx, fileID)
	if err != nil {
		return nil, err
	}
	defer downloadStream.Close()

	return io.ReadAll(downloadStream)
}

// GridFSDelete removes a file from GridFS by its ObjectID.
func GridFSDelete(ctx context.Context, fileID bson.ObjectID) error {
	bucket := database.GridFSBucket()
	return bucket.Delete(ctx, fileID)
}
