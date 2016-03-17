Uploader
========

Command line tool that receives a folder as parameter and then iterates recursively on all folders and files and creates an incident / investigation for each top folder and a table for each sub folder with the files (paths) and relevant metadata like md5, sha1, sha256, sha512 and ssdeep.

Expected folder structure is:

```
IMAGE-NAME
   |
   --- folder
   |     |
   |     ---file
   |     |
   |     ---file
   |     |
   |     ...
   |
```
