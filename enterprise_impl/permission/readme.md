我的理解是这样的，我们的每个角色都定义了有哪些权限点
例如目录的开发者角色他包含了

table:read
table:write
table:update
table:delete

form:read

form:write

chart:read
directory:write
directory:read

那么目录如果是这样的
/a/b/c
然后我给c目录赋权了目录的开发者权限
然后 我点击d函数的时候，然后你发现其实我已经拥有了父亲目录的开发者角色了，然后父亲目录的开发者角色（有缓存，可以快速找到有哪些权限点），我们可以根据当前函数所需的权限点去父亲目录的拥有的权限点查询（继承），如果存在就放行，这样就可以了，不知道你为啥搞的这么复杂，真的离谱
/a/b/c/d