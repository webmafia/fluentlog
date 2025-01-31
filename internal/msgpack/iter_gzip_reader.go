package msgpack

// // GzipReader provides a streaming decompression reader for MessagePack bin values compressed with Gzip.
// func (iter *Iterator) GzipReader() io.Reader {
// 	iter.remain = iter.Len()
// 	return gzipReader{iter: iter}
// }

// type gzipReader struct {
// 	iter *Iterator
// }

// // Read implements io.Reader.
// func (g gzipReader) Read(p []byte) (int, error) {
// 	if g.iter.remain == 0 {
// 		return 0, io.EOF
// 	}

// 	// 1. Ensure the Gzip header is fully skipped before reading.
// 	if err := g.skipGzipHeader(); err != nil {
// 		return 0, err
// 	}

// 	// 2. Create a Deflate reader (no allocations)
// 	deflateReader := flate.NewReader(g.iter.BinReader())

// 	for {
// 		// 3. Ensure we don’t exceed remaining bytes
// 		if g.iter.remain < len(p) {
// 			p = p[:g.iter.remain]
// 		}

// 		// 4. Read decompressed data
// 		n, err := deflateReader.Read(p)

// 		// 5. Update remaining bytes count
// 		g.iter.remain -= n
// 		g.iter.tot += n

// 		// 6. Handle EOF and possible multistream continuation
// 		if err == io.EOF {
// 			// Skip the Gzip trailer
// 			if err := g.skipGzipTrailer(); err != nil {
// 				return n, err
// 			}

// 			// Check for another Gzip stream (multistream support)
// 			buf, err := g.iter.Peek(3)
// 			if err == nil && len(buf) >= 3 && buf[0] == 0x1F && buf[1] == 0x8B && buf[2] == 8 {
// 				// Found another Gzip stream, reset and continue reading
// 				if err := g.skipGzipHeader(); err != nil {
// 					return n, err
// 				}
// 				deflateReader.(flate.Resetter).Reset(g.iter, nil)
// 				continue // Restart reading from new stream
// 			}

// 			// No more streams, return EOF
// 			return n, io.EOF
// 		}

// 		return n, err
// 	}
// }

// // skipGzipHeader ensures we start reading at the correct Deflate position.
// func (g gzipReader) skipGzipHeader() error {
// 	if g.iter.remain < 10 {
// 		return io.ErrUnexpectedEOF
// 	}

// 	// Read the fixed Gzip header (10 bytes)
// 	buf, err := g.iter.Peek(10)
// 	if err != nil {
// 		return err
// 	}

// 	// Validate magic number and compression method
// 	if buf[0] != 0x1F || buf[1] != 0x8B || buf[2] != 8 {
// 		return fmt.Errorf("invalid Gzip header")
// 	}

// 	flags := buf[3]

// 	// 🔹 **Skip Fixed 10-Byte Header**
// 	if err := g.skipBytes(10); err != nil {
// 		return err
// 	}

// 	// 🔹 **Skip Extra Fields (Optional)**
// 	if flags&4 != 0 {
// 		if err := g.skipBytes(2); err != nil { // Read extra field length (2 bytes)
// 			return err
// 		}
// 		extraLenBuf, err := g.iter.Peek(2)
// 		if err != nil {
// 			return err
// 		}
// 		extraLen := int(extraLenBuf[0]) | int(extraLenBuf[1])<<8
// 		if err := g.skipBytes(extraLen); err != nil {
// 			return err
// 		}
// 	}

// 	// 🔹 **Skip Filename (Optional)**
// 	if flags&8 != 0 {
// 		if err := g.skipNullTerminated(); err != nil {
// 			return err
// 		}
// 	}

// 	// 🔹 **Skip Comment (Optional)**
// 	if flags&16 != 0 {
// 		if err := g.skipNullTerminated(); err != nil {
// 			return err
// 		}
// 	}

// 	// 🔹 **Skip Header CRC (Optional)**
// 	if flags&2 != 0 {
// 		if err := g.skipBytes(2); err != nil {
// 			return err
// 		}
// 	}

// 	// 🔹 **Final sanity check before decompression**
// 	if g.iter.remain < 1 {
// 		return io.ErrUnexpectedEOF
// 	}

// 	return nil
// }

// // skipGzipTrailer skips the Gzip CRC-32 and input size trailer.
// func (g gzipReader) skipGzipTrailer() error {
// 	if g.iter.remain < 8 {
// 		return io.ErrUnexpectedEOF
// 	}
// 	return g.skipBytes(8)
// }

// // skipNullTerminated skips a null-terminated string in the Gzip header.
// func (g gzipReader) skipNullTerminated() error {
// 	for {
// 		b, err := g.iter.BinReader().ReadByte() // Directly read bytes
// 		if err != nil {
// 			return err
// 		}
// 		if b == 0 {
// 			break
// 		}
// 	}
// 	return nil
// }

// func (g gzipReader) skipBytes(n int) error {
// 	g.iter.skipBytes(8)
// 	return g.iter.err
// }
