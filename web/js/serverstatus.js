// serverstatus.js. big data boom today.
var error = 0;
var d = 0;
var server_status = new Array();

function timeSince(date) {
    if (date == 0) return "从未.";
    var seconds = Math.floor((new Date() - date) / 1000);
    var interval = Math.floor(seconds / 60);
    return interval > 1 ? interval + " 分钟前." : "几秒前.";
}

function bytesToSize(bytes, precision, si = false) {
    const units = si ? ['B', 'KB', 'MB', 'GB', 'TB'] : ['B', 'KiB', 'MiB', 'GiB', 'TiB'];
    if (bytes === 0) return '0 B';
    const k = si ? 1000 : 1024;
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(precision)) + ' ' + units[i];
}

function uptime() {
    $.getJSON("json/stats.json", function(result) {
        $("#loading-notice").remove();
        if (result.reload) setTimeout(location.reload, 1000);

        result.servers.forEach((server, i) => {
            let TableRow = $("#servers tr#r" + i);
            let MableRow = $("#monitors tr#r" + i);
            let ExpandRow = $("#servers #rt" + i);
            let hack = i % 2 ? "odd" : "even";

            if (!TableRow.length) {
                $("#servers").append(
                    `<tr id="r${i}" data-toggle="collapse" data-target="#rt${i}" class="accordion-toggle ${hack}">
                        <td id="online_status"><div class="progress"><div style="width: 100%;" class="progress-bar progress-bar-warning"><small>加载中</small></div></div></td>
                        <td id="month_traffic"><div class="progress"><div style="width: 100%;" class="progress-bar progress-bar-warning"><small>加载中</small></div></div></td>
                        <td id="name">加载中</td>
                        <td id="type">加载中</td>
                        <td id="location">加载中</td>
                        <td id="uptime">加载中</td>
                        <td id="load">加载中</td>
                        <td id="network">加载中</td>
                        <td id="traffic">加载中</td>
                        <td id="cpu"><div class="progress"><div style="width: 100%;" class="progress-bar progress-bar-warning"><small>加载中</small></div></div></td>
                        <td id="memory"><div class="progress"><div style="width: 100%;" class="progress-bar progress-bar-warning"><small>加载中</small></div></div></td>
                        <td id="hdd"><div class="progress"><div style="width: 100%;" class="progress-bar progress-bar-warning"><small>加载中</small></div></div></td>
                        <td id="ping"><div class="progress"><div style="width: 100%;" class="progress-bar progress-bar-warning"><small>加载中</small></div></div></td>
                    </tr>
                    <tr class="expandRow ${hack}"><td colspan="16"><div class="accordian-body collapse" id="rt${i}">
                        <div id="expand_mem">加载中</div>
                        <div id="expand_hdd">加载中</div>
                        <div id="expand_tupd">加载中</div>
                        <div id="expand_ping">加载中</div>
                    </div></td></tr>`
                );
                TableRow = $("#servers tr#r" + i);
                ExpandRow = $("#servers #rt" + i);
                server_status[i] = true;
            }

            if (!MableRow.length) {
                $("#monitors").append(
                    `<tr id="r${i}" data-target="#rt${i}" class="accordion-toggle ${hack}">
                        <td id="monitor_status"><div class="progress"><div style="width: 100%;" class="progress-bar progress-bar-warning"><small>加载中</small></div></div></td>
                        <td id="monitor_node">加载中</td>
                        <td id="monitor_location">加载中</td>
                        <td id="monitor_text">加载中</td>
                    </tr>`
                );
                MableRow = $("#monitors tr#r" + i);
            }

            if (error) {
                TableRow.attr("data-target", "#rt" + i);
                MableRow.attr("data-target", "#rt" + i);
                server_status[i] = true;
            }

            const statusClass = server.online4 || server.online6 ? "progress-bar progress-bar-success" : "progress-bar progress-bar-danger";
            const statusText = server.online4 && server.online6 ? "双栈" : server.online4 ? "IPv4" : server.online6 ? "IPv6" : "关闭";

            TableRow.find("#online_status .progress-bar").attr("class", statusClass).html(`<small>${statusText}</small>`);
            MableRow.find("#monitor_status .progress-bar").attr("class", statusClass).html(`<small>${statusText}</small>`);

            TableRow.find("#name").html(server.name);
            MableRow.find("#monitor_node").html(server.name);
            TableRow.find("#type").html(server.type);
            TableRow.find("#location").html(server.location);
            MableRow.find("#monitor_location").html(server.location);

            if (!server.online4 && !server.online6) {
                if (server_status[i]) {
                    TableRow.find("#uptime").html("–");
                    TableRow.find("#load").html("–");
                    TableRow.find("#network").html("–");
                    TableRow.find("#traffic").html("–");
                    TableRow.find("#month_traffic .progress-bar").attr("class", "progress-bar progress-bar-warning").html("<small>关闭</small>");
                    TableRow.find("#cpu .progress-bar").attr("class", "progress-bar progress-bar-danger").css("width", "100%").html("<small>关闭</small>");
                    TableRow.find("#memory .progress-bar").attr("class", "progress-bar progress-bar-danger").css("width", "100%").html("<small>关闭</small>");
                    TableRow.find("#hdd .progress-bar").attr("class", "progress-bar progress-bar-danger").css("width", "100%").html("<small>关闭</small>");
                    TableRow.find("#ping .progress-bar").attr("class", "progress-bar progress-bar-danger").css("width", "100%").html("<small>关闭</small>");
                    MableRow.find("#monitor_text").html("-");
                    if (ExpandRow.hasClass("in")) ExpandRow.collapse("hide");
                    TableRow.attr("data-target", "");
                    MableRow.attr("data-target", "");
                    server_status[i] = false;
                }
            } else {
                if (!server_status[i]) {
                    TableRow.attr("data-target", "#rt" + i);
                    MableRow.attr("data-target", "#rt" + i);
                    server_status[i] = true;
                }

                const trafficdiff_in = server.network_in - server.last_network_in;
                const trafficdiff_out = server.network_out - server.last_network_out;
                const monthtraffic = `${bytesToSize(trafficdiff_in, 1, true)} | ${bytesToSize(trafficdiff_out, 1, true)}`;
                TableRow.find("#month_traffic .progress-bar").attr("class", "progress-bar progress-bar-success").html(`<small>${monthtraffic}</small>`);

                TableRow.find("#uptime").html(server.uptime);
                TableRow.find("#load").html(server.load_1 == -1 ? "–" : server.load_1.toFixed(2));

                const netstr = `${bytesToSize(server.network_rx, 1, true)} | ${bytesToSize(server.network_tx, 1, true)}`;
                TableRow.find("#network").html(netstr);

                const trafficstr = `${bytesToSize(server.network_in, 1, true)} | ${bytesToSize(server.network_out, 1, true)}`;
                TableRow.find("#traffic").html(trafficstr);

                const cpuClass = server.cpu >= 90 ? "progress-bar progress-bar-danger" : server.cpu >= 80 ? "progress-bar progress-bar-warning" : "progress-bar progress-bar-success";
                TableRow.find("#cpu .progress-bar").attr("class", cpuClass).css("width", `${server.cpu}%`).html(`${server.cpu}%`);

                const Mem = ((server.memory_used / server.memory_total) * 100).toFixed(0);
                const memClass = Mem >= 90 ? "progress-bar progress-bar-danger" : Mem >= 80 ? "progress-bar progress-bar-warning" : "progress-bar progress-bar-success";
                TableRow.find("#memory .progress-bar").attr("class", memClass).css("width", `${Mem}%`).html(`${Mem}%`);
                ExpandRow.find("#expand_mem").html(`内存|虚存: ${bytesToSize(server.memory_used * 1024, 1)} / ${bytesToSize(server.memory_total * 1024, 1)} | ${bytesToSize(server.swap_used * 1024, 0)} / ${bytesToSize(server.swap_total * 1024, 0)}`);

                const HDD = ((server.hdd_used / server.hdd_total) * 100).toFixed(0);
                const hddClass = HDD >= 90 ? "progress-bar progress-bar-danger" : HDD >= 80 ? "progress-bar progress-bar-warning" : "progress-bar progress-bar-success";
                TableRow.find("#hdd .progress-bar").attr("class", hddClass).css("width", `${HDD}%`).html(`${HDD}%`);
                const io = `${bytesToSize(server.io_read, 0, true)} / ${bytesToSize(server.io_write, 0, true)}`;
                ExpandRow.find("#expand_hdd").html(`硬盘|读写: ${bytesToSize(server.hdd_used * 1024 * 1024, 2)} / ${bytesToSize(server.hdd_total * 1024 * 1024, 2)} | ${io}`);

                ExpandRow.find("#expand_tupd").html(`TCP/UDP/进/线: ${server.tcp_count} / ${server.udp_count} / ${server.process_count} / ${server.thread_count}`);

                const PING_10010 = server.ping_10010.toFixed(0);
                const PING_189 = server.ping_189.toFixed(0);
                const PING_10086 = server.ping_10086.toFixed(0);
                const pingClass = PING_10010 >= 20 || PING_189 >= 20 || PING_10086 >= 20 ? "progress-bar progress-bar-danger" : PING_10010 >= 10 || PING_189 >= 10 || PING_10086 >= 10 ? "progress-bar progress-bar-warning" : "progress-bar progress-bar-success";
                TableRow.find("#ping .progress-bar").attr("class", pingClass).html(`${PING_10010}%💻${PING_189}%💻${PING_10086}%`);
                ExpandRow.find("#expand_ping").html(`CU/CT/CM: ${server.time_10010}ms (${PING_10010}%) / ${server.time_189}ms (${PING_189}%) / ${server.time_10086}ms (${PING_10086}%)`);

                MableRow.find("#monitor_text").html(server.custom);
            }
        });

        d = new Date(result.updated * 1000);
        error = 0;
    }).fail(function() {
        if (!error) {
            $("#servers > tr.accordion-toggle").each(function(i) {
                const TableRow = $("#servers tr#r" + i);
                const MableRow = $("#monitors tr#r" + i);
                const ExpandRow = $("#servers #rt" + i);
                TableRow.find(".progress-bar").attr("class", "progress-bar progress-bar-error").html("<small>错误</small>");
                MableRow.find(".progress-bar").attr("class", "progress-bar progress-bar-error").html("<small>错误</small>");
                if (ExpandRow.hasClass("in")) ExpandRow.collapse("hide");
                TableRow.attr("data-target", "");
                MableRow.attr("data-target", "");
                server_status[i] = false;
            });
        }
        error = 1;
        $("#updated").html("更新错误.");
    });
}

function updateTime() {
    if (!error) $("#updated").html("最后更新: " + timeSince(d));
}

uptime();
updateTime();
setInterval(uptime, 2000);
setInterval(updateTime, 2000);

// styleswitcher.js
function setActiveStyleSheet(title, cookie = false) {
    Array.from(document.getElementsByTagName("link")).forEach(a => {
        if (a.getAttribute("rel").includes("style") && a.getAttribute("title")) {
            a.disabled = a.getAttribute("title") !== title;
        }
    });
    if (cookie) createCookie("style", title, 365);
}

function getActiveStyleSheet() {
    return Array.from(document.getElementsByTagName("link")).find(a => a.getAttribute("rel").includes("style") && a.getAttribute("title") && !a.disabled)?.getAttribute("title") || null;
}

function createCookie(name, value, days) {
    const expires = days ? `; expires=${new Date(Date.now() + days * 24 * 60 * 60 * 1000).toGMTString()}` : "";
    document.cookie = `${name}=${value}${expires}; path=/`;
}

function readCookie(name) {
    const nameEQ = name + "=";
    const ca = document.cookie.split(';');
    for (let i = 0; i < ca.length; i++) {
        let c = ca[i];
        while (c.charAt(0) == ' ') c = c.substring(1, c.length);
        if (c.indexOf(nameEQ) == 0) return c.substring(nameEQ.length, c.length);
    }
    return null;
}

window.onload = function() {
    const cookie = readCookie("style");
    if (cookie && cookie != 'null') {
        setActiveStyleSheet(cookie);
    } else {
        const handleChange = mediaQueryListEvent => setActiveStyleSheet(mediaQueryListEvent.matches ? 'dark' : 'light');
        const mediaQueryListDark = window.matchMedia('(prefers-color-scheme: dark)');
        setActiveStyleSheet(mediaQueryListDark.matches ? 'dark' : 'light');
        mediaQueryListDark.addEventListener("change", handleChange);
    }
}