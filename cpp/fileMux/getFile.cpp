#include <iostream>
#include <fstream>
#include <string>
#include <vector>
#include <stdexcept>
#include <cstdint>
#include <map>

// Detect video file type by magic numbers
std::string detectVideoType(const std::string &path) {
    std::ifstream file(path, std::ios::binary);
    if (!file.is_open())
        throw std::runtime_error("Cannot open file");
    std::vector<uint8_t> header(16);
    file.read(reinterpret_cast<char*>(header.data()), header.size());
    // MP4/MOV: ftyp at offset 4
    if (header.size() >= 8 && std::string(header.begin()+4, header.begin()+8) == "ftyp")
        return "MP4/MOV";
    // MKV/WebM: 1A 45 DF A3
    if (header.size() >= 4 && header[0] == 0x1A && header[1] == 0x45 && header[2] == 0xDF && header[3] == 0xA3)
        return "MKV/WebM";
    // AVI: RIFF....AVI
    if (header.size() >= 12 && std::string(header.begin(), header.begin()+4) == "RIFF" && std::string(header.begin()+8, header.begin()+12) == "AVI ")
        return "AVI";
    // FLV: FLV
    if (header.size() >= 3 && std::string(header.begin(), header.begin()+3) == "FLV")
        return "FLV";
    // MPEG: 00 00 01 BA
    if (header.size() >= 4 && header[0] == 0x00 && header[1] == 0x00 && header[2] == 0x01 && header[3] == 0xBA)
        return "MPEG";
    // QuickTime: moov at offset 4
    if (header.size() >= 8 && std::string(header.begin()+4, header.begin()+8) == "moov")
        return "QuickTime";
    // Others can be added here
    return "Unknown";
}

// Struct to hold file stream and detected type
struct FileWithType {
    std::ifstream file;
    std::string fileType;
    FileWithType(const std::string &path) : file(path, std::ios::binary) {
        if (!file.is_open())
            throw std::runtime_error("Cannot open file");
        fileType = detectVideoType(path);
    }
};

// 1. Input taker function
FileWithType openFile(const std::string &path)
{
    return FileWithType(path);
}

// 2. Demuxer function (super simple stub, just reads raw bytes for now)
std::vector<uint8_t> demux(std::ifstream &file)
{
    std::vector<uint8_t> buffer(
        (std::istreambuf_iterator<char>(file)),
        (std::istreambuf_iterator<char>()));
    // later this will actually split into packets
    return buffer;
}

// 3. Main function
int main(int argc, char **argv)
{
    if (argc < 2)
    {
        std::cerr << "Usage: mydemux <file>" << std::endl;
        return -1;
    }
    auto fileWithType = openFile(argv[1]);
    auto data = demux(fileWithType.file);
    std::cout << "Read " << data.size() << " bytes" << std::endl;
    // give me cout of everything data has inside
    for (const auto &byte : data) {
        std::cout << std::hex << static_cast<int>(byte) << " ";
    }
    std::cout << std::dec << std::endl; // reset to decimal output

    std::cout << "Detected file type: " << fileWithType.fileType << std::endl;
    return 0;
}
