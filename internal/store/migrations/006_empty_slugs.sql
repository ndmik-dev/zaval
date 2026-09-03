-- Projects named in Cyrillic ended up with an empty ⌘K tag.
update projects set slug = 'p' || id where slug = '';
