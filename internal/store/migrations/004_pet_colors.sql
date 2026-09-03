-- Pet projects that still carry the shared default ochre get distinct colors.
update projects set color = case (select count(*) from projects p2 where p2.kind = 'pet' and p2.id < projects.id) % 6
  when 0 then '#8A6A30' when 1 then '#4E6B8C' when 2 then '#7A5C99'
  when 3 then '#3E7D6E' when 4 then '#A0522D' else '#5C7A3E' end
where kind = 'pet' and color = '#8A6A30';
