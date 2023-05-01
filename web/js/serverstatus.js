
var $ = layui.jquery, layer = layui.layer, element = layui.element;
// serverstatus.js. big data boom today.
var error = 0;
var d = 0;
var server_status = new Array();

function timeSince(date) {
	if (date == 0)
		return "从未.";

	var seconds = Math.floor((new Date() - date) / 1000);
	var interval = Math.floor(seconds / 60);
	if (interval > 1)
		return interval + " 分钟前.";
	else
		return "几秒前.";
}

function bytesToSize(bytes, precision, si) {
	var ret;
	si = typeof si !== 'undefined' ? si : 0;
	if (si != 0) {
		var megabyte = 1000 * 1000;
		var gigabyte = megabyte * 1000;
		var terabyte = gigabyte * 1000;
	} else {
		var megabyte = 1024 * 1024;
		var gigabyte = megabyte * 1024;
		var terabyte = gigabyte * 1024;
	}

	if ((bytes >= megabyte) && (bytes < gigabyte)) {
		ret = (bytes / megabyte).toFixed(precision) + ' M';

	} else if ((bytes >= gigabyte) && (bytes < terabyte)) {
		ret = (bytes / gigabyte).toFixed(precision) + ' G';

	} else if (bytes >= terabyte) {
		ret = (bytes / terabyte).toFixed(precision) + ' T';

	} else {
		return bytes + ' B';
	}
	return ret;
	/*if(si != 0) {
		return ret + 'B';
	} else {
		return ret + 'iB';
	}*/
}

$.fn.alterClass = function (removals, additions) {

	var self = this;

	if (removals.indexOf('*') === -1) {
		// Use native jQuery methods if there is no wildcard matching
		self.removeClass(removals);
		return !additions ? self : self.addClass(additions);
	}

	var patt = new RegExp('\\s' +
		removals.
			replace(/\*/g, '[A-Za-z0-9-_]+').
			split(' ').
			join('\\s|\\s') +
		'\\s', 'g');

	self.each(function (i, it) {
		var cn = ' ' + it.className + ' ';
		while (patt.test(cn)) {
			cn = cn.replace(patt, ' ');
		}
		it.className = $.trim(cn);
	});

	return !additions ? self : self.addClass(additions);
};

function uptime() {
	$.getJSON("json/stats.json", function (result) {
		$("#loading-notice").remove();
		if (result.reload)
			setTimeout(function () { location.reload() }, 1000);

		for (var i = 0, rlen = result.servers.length; i < rlen; i++) {
			var TableRow = $("#servers tr#r" + i);
			var ExpandRow = $("#servers #rt" + i);
			var hack; // fuck CSS for making me do this
			if (i % 2) hack = "odd"; else hack = "even";
			if (!TableRow.length) {
				$("#servers").append(
					$("<tr>", {
						id: "r" + i,
						"data-target": "#rt" + i,
						class: "accordion-toggle " + hack,
						html: "<td id='online_status'><div class='layui-badge'>加载中</div></td>" +
							"<td id='month_traffic'><div class='layui-badge'>加载中</div></td>" +
							"<td id='name'>加载中</td>" +
							"<td id='type'>加载中</td>" +
							"<td id='location'>加载中</td>" +
							"<td id='uptime'>加载中</td>" +
							"<td id='load'>加载中</td>" +
							"<td id='network'>加载中</td>" +
							"<td id='traffic'>加载中</td>" +
							"<td id='cpu'><div class='layui-progress layui-progress-big' lay-showpercent='true'><div style='width: 100%;' class='layui-progress-bar layui-bg-orange'><small>加载中</small></div></div></td>" +
							"<td id='memory'><div class='layui-progress layui-progress-big' lay-showpercent='true'><div style='width: 100%;' class='layui-progress-bar layui-bg-orange'><small>加载中</small></div></div></td>" +
							"<td id='hdd'><div class='layui-progress layui-progress-big' lay-showpercent='true'><div style='width: 100%;' class='layui-progress-bar layui-bg-orange'><small>加载中</small></div></div></td>" +
							"<td id='ping'><div class='layui-badge'>加载中</div></td>"
					}),
					$("<tr>", {
						class: "expandRow " + hack,
						html: "<td colspan='16' style='display: none;'><div class='layui-text collapse' id='rt" + i + "'>" +
							"<div id='expand_mem'>加载中</div>" +
							"<div id='expand_hdd'>加载中</div>" +
							"<div id='expand_tupd'>加载中</div>" +
							"<div id='expand_ping'>加载中</div>" +
							"<div id='expand_lost'>加载中</div>" +
							"<div id='expand_custom'>加载中</div>" +
							"</div></td>"
					})
				);
				TableRow = $("#servers tr#r" + i);
				ExpandRow = $("#servers #rt" + i);
				server_status[i] = true;
			}
			TableRow = TableRow[0];
			if (error) {
				TableRow.setAttribute("data-target", "#rt" + i);
				server_status[i] = true;
			}

			// online_status
			if (result.servers[i].online4 && !result.servers[i].online6) {
				$(TableRow).find("#online_status .layui-badge").alterClass("layui-bg-*").addClass("layui-bg-green").text("IPv4");
			} else if (result.servers[i].online4 && result.servers[i].online6) {
				$(TableRow).find("#online_status .layui-badge").alterClass("layui-bg-*").addClass("layui-bg-green").text("双栈");
			} else if (!result.servers[i].online4 && result.servers[i].online6) {
				$(TableRow).find("#online_status .layui-badge").alterClass("layui-bg-*").addClass("layui-bg-green").text("IPv6");
			} else {
				$(TableRow).find("#online_status .layui-badge").alterClass("layui-bg-*").addClass("layui-bg-red").text("关闭");
			}

			// Name
			$(TableRow).find("#name").text(result.servers[i].name);

			// Type
			$(TableRow).find("#type").text(result.servers[i].type);

			// Location
			$(TableRow).find("#location").text(result.servers[i].location);
			if (!result.servers[i].online4 && !result.servers[i].online6) {
				if (server_status[i]) {
					$(TableRow).find("#uptime").text("–");
					$(TableRow).find("#load").text("–");
					$(TableRow).find("#network").text("–");
					$(TableRow).find("#traffic").text("–");
					$(TableRow).find("#month_traffic .layui-badge").alterClass("layui-bg-*").addClass("layui-bg-orange").text("关闭");
					$(TableRow).find("#cpu .layui-progress-bar").alterClass("layui-bg-*").addClass("layui-bg-red").css("width", "100%").html('<span class="layui-progress-text">关闭</span>');
					$(TableRow).find("#memory .layui-progress-bar").alterClass("layui-bg-*").addClass("layui-bg-red").css("width", "100%").html('<span class="layui-progress-text">关闭</span>');
					$(TableRow).find("#hdd .layui-progress-bar").alterClass("layui-bg-*").addClass("layui-bg-red").css("width", "100%").html('<span class="layui-progress-text">关闭</span>');
					$(TableRow).find("#ping .layui-badge").alterClass("layui-bg-*").addClass("layui-bg-red").text("关闭");
					$($(TableRow).attr("data-target")).parent().slideUp();
					$(TableRow).attr("data-target", "");
					server_status[i] = false;
				}
			} else {
				if (!server_status[i]) {
					$(TableRow).attr("data-target", "#rt" + i);
					server_status[i] = true;
				}

				// month traffic
				var monthtraffic = "";
				var trafficdiff_in = result.servers[i].network_in - result.servers[i].last_network_in;
				var trafficdiff_out = result.servers[i].network_out - result.servers[i].last_network_out;
				if (trafficdiff_in < 1024 * 1024 * 1024 * 1024)
					monthtraffic += (trafficdiff_in / 1024 / 1024 / 1024).toFixed(1) + "G";
				else
					monthtraffic += (trafficdiff_in / 1024 / 1024 / 1024 / 1024).toFixed(1) + "T";
				monthtraffic += " | "
				if (trafficdiff_out < 1024 * 1024 * 1024 * 1024)
					monthtraffic += (trafficdiff_out / 1024 / 1024 / 1024).toFixed(1) + "G";
				else
					monthtraffic += (trafficdiff_out / 1024 / 1024 / 1024 / 1024).toFixed(1) + "T";
				$(TableRow).find("#month_traffic .layui-badge").alterClass("layui-bg-*").addClass("layui-bg-green").html("<small>" + monthtraffic + "</small>");

				// Uptime
				$(TableRow).find("#uptime").text(result.servers[i].uptime);

				// Load: default load_1, you can change show: load_1, load_5, load_15
				if (result.servers[i].load == -1) {
					$(TableRow).find("#load").text("–");
				} else {
					$(TableRow).find("#load").text(result.servers[i].load_1.toFixed(2));
				}

				// Network
				var netstr = "";
				if (result.servers[i].network_rx < 1024 * 1024)
					netstr += (result.servers[i].network_rx / 1024).toFixed(1) + "K";
				else
					netstr += (result.servers[i].network_rx / 1024 / 1024).toFixed(1) + "M";
				netstr += " | "
				if (result.servers[i].network_tx < 1024 * 1024)
					netstr += (result.servers[i].network_tx / 1024).toFixed(1) + "K";
				else
					netstr += (result.servers[i].network_tx / 1024 / 1024).toFixed(1) + "M";
				$(TableRow).find("#network").text(netstr);

				//Traffic
				var trafficstr = "";
				if (result.servers[i].network_in < 1024 * 1024 * 1024 * 1024)
					trafficstr += (result.servers[i].network_in / 1024 / 1024 / 1024).toFixed(1) + "G";
				else
					trafficstr += (result.servers[i].network_in / 1024 / 1024 / 1024 / 1024).toFixed(1) + "T";
				trafficstr += " | "
				if (result.servers[i].network_out < 1024 * 1024 * 1024 * 1024)
					trafficstr += (result.servers[i].network_out / 1024 / 1024 / 1024).toFixed(1) + "G";
				else
					trafficstr += (result.servers[i].network_out / 1024 / 1024 / 1024 / 1024).toFixed(1) + "T";
				$(TableRow).find("#traffic").text(trafficstr);

				// CPU
				if (result.servers[i].cpu >= 90)
					$(TableRow).find("#cpu .layui-progress-bar").alterClass("layui-bg-*").addClass("layui-bg-red");
				else if (result.servers[i].cpu >= 80)
					$(TableRow).find("#cpu .layui-progress-bar").alterClass("layui-bg-*").addClass("layui-bg-orange");
				else
					$(TableRow).find("#cpu .layui-progress-bar").alterClass("layui-bg-*").addClass("layui-bg-green");
				$(TableRow).find("#cpu .layui-progress-bar").css("width", result.servers[i].cpu + "%").html('<span class="layui-progress-text">' + result.servers[i].cpu + "%</span>");

				// Memory
				var Mem = ((result.servers[i].memory_used / result.servers[i].memory_total) * 100.0).toFixed(0);
				if (Mem >= 90)
					$(TableRow).find("#memory .layui-progress-bar").alterClass("layui-bg-*").addClass("layui-bg-red");
				else if (Mem >= 80)
					$(TableRow).find("#memory .layui-progress-bar").alterClass("layui-bg-*").addClass("layui-bg-orange");
				else
					$(TableRow).find("#memory .layui-progress-bar").alterClass("layui-bg-*").addClass("layui-bg-green");
				$(TableRow).find("#memory .layui-progress-bar").css("width", Mem + "%").html('<span class="layui-progress-text">' + Mem + "%</span>");
				// 内存|swap
				$(ExpandRow).find("#expand_mem").html("内存|虚存:" + bytesToSize(result.servers[i].memory_used * 1024, 1) + " / " + bytesToSize(result.servers[i].memory_total * 1024, 1) + " | " + bytesToSize(result.servers[i].swap_used * 1024, 0) + " / " + bytesToSize(result.servers[i].swap_total * 1024, 0));

				// HDD
				var HDD = ((result.servers[i].hdd_used / result.servers[i].hdd_total) * 100.0).toFixed(0);
				if (HDD >= 90)
					$(TableRow).find("#hdd .layui-progress-bar").alterClass("layui-bg-*").addClass("layui-bg-red");
				else if (HDD >= 80)
					$(TableRow).find("#hdd .layui-progress-bar").alterClass("layui-bg-*").addClass("layui-bg-orange");
				else
					$(TableRow).find("#hdd .layui-progress-bar").alterClass("layui-bg-*").addClass("layui-bg-green");
				$(TableRow).find("#hdd .layui-progress-bar").css("width", HDD + "%").html('<span class="layui-progress-text">' + HDD + "%</span>");
				// IO Speed for HDD.
				// IO， 过小的B字节单位没有意义
				var io = "";
				if (result.servers[i].io_read < 1024 * 1024)
					io += parseInt(result.servers[i].io_read / 1024) + "K";
				else
					io += parseInt(result.servers[i].io_read / 1024 / 1024) + "M";
				io += " / "
				if (result.servers[i].io_write < 1024 * 1024)
					io += parseInt(result.servers[i].io_write / 1024) + "K";
				else
					io += parseInt(result.servers[i].io_write / 1024 / 1024) + "M";
				// Expand for HDD.
				$(ExpandRow).find("#expand_hdd").html("硬盘|读写: " + bytesToSize(result.servers[i].hdd_used * 1024 * 1024, 2) + " / " + bytesToSize(result.servers[i].hdd_total * 1024 * 1024, 2) + " | " + io);

				// delay time

				// tcp, udp, process, thread count
				$(ExpandRow).find("#expand_tupd").html("TCP/UDP/进/线: " + result.servers[i].tcp_count + " / " + result.servers[i].udp_count + " / " + result.servers[i].process_count + " / " + result.servers[i].thread_count);
				$(ExpandRow).find("#expand_ping").html("联通/电信/移动: " + result.servers[i].time_10010 + "ms / " + result.servers[i].time_189 + "ms / " + result.servers[i].time_10086 + "ms");

				// ping
				var PING_10010 = result.servers[i].ping_10010.toFixed(0);
				var PING_189 = result.servers[i].ping_189.toFixed(0);
				var PING_10086 = result.servers[i].ping_10086.toFixed(0);
				$(ExpandRow).find("#expand_lost").html("丢包：联通/电信/移动: " + PING_10010 + "% / " + PING_189 + "% / " + PING_10086 + "%");

				if (PING_10010 >= 20 || PING_189 >= 20 || PING_10086 >= 20)
					$(TableRow).find("#ping .layui-badge").alterClass("layui-bg-*").addClass("layui-bg-red");
				else if (PING_10010 >= 10 || PING_189 >= 10 || PING_10086 >= 10)
					$(TableRow).find("#ping .layui-badge").alterClass("layui-bg-*").addClass("layui-bg-orange");
				else
					$(TableRow).find("#ping .layui-badge").alterClass("layui-bg-*").addClass("layui-bg-green");
				$(TableRow).find("#ping .layui-badge").html(PING_10010 + "%💻" + PING_189 + "%💻" + PING_10086 + "%");

				// Custom
				if (result.servers[i].custom) {
					$(ExpandRow).find("#expand_custom").html(result.servers[i].custom);
				} else {
					$(ExpandRow).find("#expand_custom").html("");
				}
			}
		}

		d = new Date(result.updated * 1000);
		error = 0;
	}).fail(function (update_error) {
		if (!error) {
			$("#servers > tr.accordion-toggle").each(function (i) {
				var TableRow = $("#servers tr#r" + i)[0];
				var ExpandRow = $("#servers #rt" + i);
				$(TableRow).find("#online_status .layui-badge").alterClass("layui-bg-*").addClass("layui-bg-red").text("错误");
				$(TableRow).find("#month_traffic .layui-badge").alterClass("layui-bg-*").addClass("layui-bg-red").text("错误");
				$(TableRow).find("#uptime").html("<div class='layui-badge'>错误</div>");
				$(TableRow).find("#load").html("<div class='layui-badge'>错误</div>");
				$(TableRow).find("#network").html("<div class='layui-badge'>错误</div>");
				$(TableRow).find("#traffic").html("<div class='layui-badge'>错误</div>");
				$(TableRow).find("#cpu .layui-progress-bar").alterClass("layui-bg-*").addClass("layui-bg-red").css("width", "100%").html('<span class="layui-progress-text">错误</span>');
				$(TableRow).find("#memory .layui-progress-bar").alterClass("layui-bg-*").addClass("layui-bg-red").css("width", "100%").html('<span class="layui-progress-text">错误</span>');
				$(TableRow).find("#hdd .layui-progress-bar").alterClass("layui-bg-*").addClass("layui-bg-red").css("width", "100%").html('<span class="layui-progress-text">错误</span>');
				$(TableRow).find("#ping .layui-badge").alterClass("layui-bg-*").addClass("layui-bg-red").text("错误");
				$($(TableRow).attr("data-target")).parent().slideUp();
				$(TableRow).attr("data-target", "");
				server_status[i] = false;
			});
		}
		error = 1;
		$("#updated").html("更新错误.");
	});

}

function updateTime() {
	if (!error)
		$("#updated").html("最后更新: " + timeSince(d));
}

uptime();
updateTime();
setInterval(uptime, 2000);
setInterval(updateTime, 2000);


function getActiveStyleSheet() {
	var i, a;
	for (i = 0; (a = document.getElementsByTagName("link")[i]); i++) {
		if (a.getAttribute("rel").indexOf("style") != -1 && a.getAttribute("title") && !a.disabled)
			return a.getAttribute("title");
	}
	return null;
}

function createCookie(name, value, days) {
	if (days) {
		var date = new Date();
		date.setTime(date.getTime() + (days * 24 * 60 * 60 * 1000));
		var expires = "; expires=" + date.toGMTString();
	}
	else expires = "";
	document.cookie = name + "=" + value + expires + "; path=/";
}

function readCookie(name) {
	var nameEQ = name + "=";
	var ca = document.cookie.split(';');
	for (var i = 0; i < ca.length; i++) {
		var c = ca[i];
		while (c.charAt(0) == ' ')
			c = c.substring(1, c.length);
		if (c.indexOf(nameEQ) == 0)
			return c.substring(nameEQ.length, c.length);
	}
	return null;
}

$(document).on('click', ".accordion-toggle", function () {
	$($(this).attr("data-target")).parent().slideToggle();
});

function changeTheme(title, cookie = false) {
	if (title == 'dark') {
		$('#layui_theme_css').prop('href', 'css/dark.css');
	} else {
		$('#layui_theme_css').prop('href', 'css/light.css');
	}
	if (true == cookie) {
		createCookie("style", title, 365);
	}
}
window.changeTheme = changeTheme;

window.onload = function (e) {
	var cookie = readCookie("style");
	if (cookie && cookie != 'null') {
		changeTheme(cookie);
	} else {
		function handleChange(mediaQueryListEvent) {
			if (mediaQueryListEvent.matches) {
				changeTheme('dark');
			} else {
				changeTheme('light');
			}
		}
		const mediaQueryListDark = window.matchMedia('(prefers-color-scheme: dark)');
		changeTheme(mediaQueryListDark.matches ? 'dark' : 'light');
		mediaQueryListDark.addEventListener("change", handleChange);
	}
}
