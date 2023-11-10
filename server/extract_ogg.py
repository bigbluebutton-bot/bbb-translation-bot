# https://datatracker.ietf.org/doc/html/rfc3533
# Format of the Ogg page header:

#  0                   1                   2                   3
#  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1| Byte
# +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
# | capture_pattern: Magic number for page start "OggS"           | 0-3
# +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
# | version       | header_type   | granule_position              | 4-7
# +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
# |                                                               | 8-11
# +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
# |                               | bitstream_serial_number       | 12-15
# +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
# |                               | page_sequence_number          | 16-19
# +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
# |                               | CRC_checksum                  | 20-23
# +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
# |                               |page_segments  | segment_table | 24-27
# +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
# | ...                                                           | 28-
# +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+

# Explanation of the fields in the page header:

# 1. capture_pattern: a 4-byte field indicating the start of a page. 
#    It contains the magic numbers:
#       0x4f 'O'
#       0x67 'g'
#       0x67 'g'
#       0x53 'S'
#    This helps a decoder find page boundaries and regain synchronization after parsing a corrupted stream. 
#    After finding the capture pattern, the decoder verifies page sync and integrity by computing and comparing the checksum.

# 2. stream_structure_version: a 1-byte field indicating the Ogg file format version used in this stream (this document specifies version 0).

# 3. header_type_flag: a 1-byte field specifying the type of this page.
#    - Continuation Flag (0x01): Bit 0. If set (1), it indicates that this page continues a packet from the previous page. 
#      If not set (0), the first packet on this page is the start of a new packet.
#    - Beginning of Stream Flag (0x02): Bit 1. If set (1), it signals the start of a stream. 
#      Typically set only on the first page of an Ogg stream.
#    - End of Stream Flag (0x04): Bit 2. If set (1), it marks the end of the stream. 
#      Set only on the last page of a logical stream.
#    - Reserved Bits (0x08 to 0x80): Bits 3 to 7 are reserved for future use. Typically set to 0.

# 4. granule_position: an 8-byte field with position information, e.g., total number of PCM samples or video frames encoded after this page. 
#    A value of -1 indicates that no packets finish on this page.

# 5. bitstream_serial_number: a 4-byte field containing the unique serial number identifying the logical bitstream.

# 6. page_sequence_number: a 4-byte field with the page's sequence number, helping the decoder identify page loss. 
#    Increments with each page in each logical bitstream.

# 7. CRC_checksum: a 4-byte field with a 32-bit CRC checksum of the page, including header with zero CRC field and page content. 
#    The generator polynomial is 0x04c11db7.

# 8. number_page_segments: a 1-byte field indicating the number of segment entries in the segment table.

# 9. segment_table: a series of bytes equal to the number_page_segments, containing the lacing values of all segments in this page.

# Total header size in bytes: header_size = number_page_segments + 27 [Byte]
# Total page size in bytes: page_size = header_size + sum(lacing_values: 1..number_page_segments) [Byte]

class OggSFrame:
    def __init__(self, frame_data, endianness="little"):
        # Check if the frame data block is in the correct format
        if len(frame_data) < 27 or frame_data[0:4] != b'OggS':
            raise ValueError("Invalid OggS frame data.")

        # Extract header information
        self.header = {
            'capture_pattern': frame_data[0:4],
            'version': frame_data[4],
            'header_type': frame_data[5],
            'granule_position': int.from_bytes(frame_data[6:14], endianness),
            'serial_number': int.from_bytes(frame_data[14:18], endianness),
            'page_sequence_number': int.from_bytes(frame_data[18:22], endianness),
            'CRC_checksum': frame_data[22:26],
            'segment_count': frame_data[26]
        }

        # Extract the segment table and calculate the total page size
        page_segments = self.header['segment_count']
        segment_table = frame_data[27:27 + page_segments]
        page_size = 27 + page_segments + sum(segment_table)

        # Extract frame data
        self.data = frame_data[27 + page_segments:page_size]

        self.raw_data = frame_data

    def __repr__(self):
        return f"OggSFrame(Header: {self.header}, Raw data length: {len(self.raw_data)}, Data length: {len(self.data)})"

# Function to split Ogg data into an array of bytes, where each element contains one OggS frame
def split_ogg_data_into_frames(ogg_data):
    frames = []
    offset = 0

    while offset < len(ogg_data):
        # Read the first 27 bytes of the header
        header = ogg_data[offset:offset + 27]
        if len(header) < 27 or header[0:4] != b'OggS':
            break  # End if no valid header is found

        # Read the number of segments
        page_segments = header[26]
        segment_table = ogg_data[offset + 27:offset + 27 + page_segments]

        # Calculate the total page size
        page_size = 27 + page_segments + sum(segment_table)

        # Extract the entire frame
        frame = ogg_data[offset:offset + page_size]
        frames.append(OggSFrame(frame))

        # Update offset to the next frame
        offset += page_size

    # Sort frames into the correct order by page_sequence_number
    sorted_frames = sorted(frames, key=lambda frame: frame.header['page_sequence_number'])
    return sorted_frames


# https://datatracker.ietf.org/doc/html/rfc7845.html#section-5.2
# Packet Organization in an Ogg Opus stream

# An Ogg Opus stream is organized as follows (see Figure 1 for an example).

#         Page 0         Pages 1 ... n        Pages (n+1) ...
#      +------------+ +---+ +---+ ... +---+ +-----------+ +---------+ +--
#      |            | |   | |   |     |   | |           | |         | |
#      |+----------+| |+-----------------+| |+-------------------+ +-----
#      |||ID Header|| ||  Comment Header || ||Audio Data Packet 1| | ...
#      |+----------+| |+-----------------+| |+-------------------+ +-----
#      |            | |   | |   |     |   | |           | |         | |
#      +------------+ +---+ +---+ ... +---+ +-----------+ +---------+ +--
#      ^      ^                           ^
#      |      |                           |
#      |      |                           Mandatory Page Break
#      |      |
#      |      ID header is contained on a single page
#      |
#      'Beginning Of Stream'

#     Figure 1: Example Packet Organization for a Logical Ogg Opus Stream

def extract_id_header_frame(frames):
    return frames[0]

def extract_comment_header_frames(frames):
    # Start with the assumption that the first frame after the ID Header contains the Comment Header
    comment_header_started = False
    comment_header_completed = False
    comment_header_frames = []

    for frame in frames:
        if not comment_header_started:
            # Look for the frame immediately following the ID Header (page_sequence_number = 1)
            if frame.header['page_sequence_number'] == 1:
                comment_header_started = True
                comment_header_completed = False
                comment_header_frames.append(frame)
        else:
            # Continue adding frames until the Comment Header is complete
            # The Comment Header completes on a page where 'header_type' does not have the 'continued packet' flag set
            if frame.header['header_type'] & 0x01 == 0:  # Checking the 'continued packet' flag
                comment_header_frames.append(frame)
                comment_header_completed = True
                break
            else:
                comment_header_frames.append(frame)

    if not comment_header_completed:
        comment_header_frames = []

    return comment_header_frames


def get_header_frames(byte_data):
    id_header_frame = None
    comment_header_frames = []

    ogg_frames = split_ogg_data_into_frames(byte_data)
    id_header_frame = extract_id_header_frame(ogg_frames)
    comment_header_frames = extract_comment_header_frames(ogg_frames)

    if id_header_frame and len(comment_header_frames) > 0:
        return id_header_frame, comment_header_frames
    else:
        return None, []




def __main__() -> None:
    # Path to the Ogg file
    from anyio import Path
    file_path = Path('sample.opus')
    with open(file_path, 'rb') as file:
        ogg_bytes = file.read()

    id_header_frame, comment_header_frames = get_header_frames(ogg_bytes)
    print("ID Header Frame:")
    print(id_header_frame)

    print("Comment Header Frames:")
    for frames in comment_header_frames:
        print(frames)


if __name__ == "__main__":
    __main__()