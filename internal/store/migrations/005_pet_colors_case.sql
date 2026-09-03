-- Same as 004, but colors saved from the picker are lowercase.
update projects set color = case (select count(*) from projects p2 where p2.kind = 'pet' and p2.id < projects.id) % 6
  when 0 then '#8A6A30' when 1 then '#4E6B8C' when 2 then '#7A5C99'
  when 3 then '#3E7D6E' when 4 then '#A0522D' else '#5C7A3E' end
where kind = 'pet' and lower(color) = '#8a6a30'
  and (select count(*) from projects p3 where p3.kind = 'pet' and lower(p3.color) = '#8a6a30') > 1;
