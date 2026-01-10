# Golang .DS_Store reader and writer
Go implemetation for reading and writing for .DS_Store files.
The implemetation based on: 
* https://0day.work/parsing-the-ds_store-file-format/
* https://metacpan.org/pod/distribution/Mac-Finder-DSStore/DSStoreFormat.pod
* https://wiki.mozilla.org/DS_Store_File_Format
* https://github.com/gehaxelt/ds_store
* https://github.com/gehaxelt/Python-dsstore

Looks as work. I use it for renaming application name in .DS_Store template when collecting DMG installer.

Parsed .DS_Store records contains the folowing fields:
* FileName - file name
* Extra - unknown 4 bytes
* Type - 4 bytes string
* DataLen - data len for some types (blob, ustr), for primitive types it must be 0.

  When value is zero DataLen is not writes to output. So zero-size blobs doesn't work.
* Data - bytes arrays of data

The full description about .DS_Store records can be found here:

https://metacpan.org/pod/distribution/Mac-Finder-DSStore/DSStoreFormat.pod#Records

The implementation just reads and writes container and don't parse Data field and etc.
Data field with "blob" type often contains binary property list (plist) and should be parsed through other modules or libraries.

Blocks allocation on writing can be have different order and size than be was read.

# WARNING
BE CAREFUL USE IT FOR WRITING .DS_Store!

Do not use it for massive modifying of .DS_Store files. Just use it for installer purpose (preparing .DS_Store for DMG).

# License

MIT
