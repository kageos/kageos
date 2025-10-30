package widget

// Link 这里要看是table函数还是form函数，table函数用这种
type Link struct {
	//[这里是超链接的标题](这里是超链接) 默认会在当前目录下面查找对应的url，例如你在vote目录
	//这个目录下有/vote/vote_get /vote/vote_list /vote/vote_update /vote/vote_post
	//你在/vote/vote_get 接口返回了 //[查看我的列表](vote_list?like=create_by:$create_by&sort=id:desc) ，会直接跳转到当前目录的/vote/vote_list 这个接口下，
	//假如是//[权限申请](/auth/apply/user) 这种，明显当前目录不存在，前端会直接用host+url来实现绝对跳转，跳转到其他目录或者其他的服务目录下面

	//跳转到table
	Url string `json:"url"` //[查看我的列表](vote_list?like=create_by:$create_by&sort=id:desc)
	//in=education:高中,大专  这里是查询高中和大专的记录
	//gte=graduation_year:2019 毕业年份大雨等于2019的记录
	//like=name:beiluo 名字包含beiluo的记录
	//gte=graduation_year:2019&in=education:高中,大专  查询2019年及其以后高中或者大专毕业的记录
	//gte=graduation_year:$graduation_year  查询大于当前这条记录graduation_year的记录
	//eq=id:$id  查询当前这条记录的信息

	//上面的这种属于是跳转到table的模版才需要用search标签里的那种方式，不在search标签里的不支持搜索，所以?后面要跟标签对齐

	//跳转到form的话
	//可以更简单了，直接用正常的url链接即可
	//Url string `json:"url"` //[去投票](vote_post?id=$vote_id)

	Type string `json:"type"` //to_table,to_form

}
