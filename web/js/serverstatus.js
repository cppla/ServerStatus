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
    fetch("json/stats.json")
        .then(response => {
            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }
            return response.json();
        })
        .then(result => {
            document.getElementById("loading-notice")?.remove();
            if (result.reload) setTimeout(location.reload, 1000);

            result.servers.forEach((server, i) => {
                let TableRow = document.querySelector(`#servers tr#r${i}`);
                let MableRow = document.querySelector(`#monitors tr#r${i}`);
                let ExpandRow = document.querySelector(`#servers #rt${i}`);
                let hack = i % 2 ? "odd" : "even";

                if (!TableRow) {
                    document.getElementById("servers").insertAdjacentHTML(
                        "beforeend",
                        `<tr id="r${i}" data-bs-toggle="collapse" data-bs-target="#rt${i}" class="accordion-toggle ${hack}">
                            <td id="online_status"><div class="progress"><div style="width: 100%;" class="progress-bar bg-warning"><small>加载中</small></div></div></td>
                            <td id="month_traffic"><div class="progress"><div style="width: 100%;" class="progress-bar bg-warning"><small>加载中</small></div></div></td>
                            <td id="name">加载中</td>
                            <td id="type">加载中</td>
                            <td id="location">加载中</td>
                            <td id="uptime">加载中</td>
                            <td id="load">加载中</td>
                            <td id="network">加载中</td>
                            <td id="traffic">加载中</td>
                            <td id="cpu"><div class="progress"><div style="width: 100%;" class="progress-bar bg-warning"><small>加载中</small></div></div></td>
                            <td id="memory"><div class="progress"><div style="width: 100%;" class="progress-bar bg-warning"><small>加载中</small></div></div></td>
                            <td id="hdd"><div class="progress"><div style="width: 100%;" class="progress-bar bg-warning"><small>加载中</small></div></div></td>
                            <td id="ping"><div class="progress"><div style="width: 100%;" class="progress-bar bg-warning"><small>加载中</small></div></div></td>
                        </tr>
                        <tr class="expandRow ${hack}"><td colspan="16"><div class="accordian-body collapse" id="rt${i}">
                            <div id="expand_mem">加载中</div>
                            <div id="expand_hdd">加载中</div>
                            <div id="expand_tupd">加载中</div>
                            <div id="expand_ping">加载中</div>
                        </div></td></tr>`
                    );
                    TableRow = document.querySelector(`#servers tr#r${i}`);
                    ExpandRow = document.querySelector(`#servers #rt${i}`);
                    server_status[i] = true;
                }

                if (!MableRow) {
                    document.getElementById("monitors").insertAdjacentHTML(
                        "beforeend",
                        `<tr id="r${i}" data-bs-target="#rt${i}" class="accordion-toggle ${hack}">
                            <td id="monitor_status"><div class="progress"><div style="width: 100%;" class="progress-bar bg-warning"><small>加载中</small></div></div></td>
                            <td id="monitor_node">加载中</td>
                            <td id="monitor_location">加载中</td>
                            <td id="monitor_text">加载中</td>
                        </tr>`
                    );
                    MableRow = document.querySelector(`#monitors tr#r${i}`);
                }

                if (error) {
                    TableRow.setAttribute("data-bs-target", `#rt${i}`);
                    MableRow.setAttribute("data-bs-target", `#rt${i}`);
                    server_status[i] = true;
                }

                const statusClass = server.online4 || server.online6 ? "progress-bar bg-success" : "progress-bar bg-danger";
                const statusText = server.online4 && server.online6 ? "双栈" : server.online4 ? "IPv4" : server.online6 ? "IPv6" : "关闭";

                if (TableRow) {
                    const onlineStatusBar = TableRow.querySelector("#online_status .progress-bar");
                    if (onlineStatusBar) {
                        onlineStatusBar.setAttribute("class", statusClass);
                        onlineStatusBar.innerHTML = `<small>${statusText}</small>`;
                    }
                }

                if (MableRow) {
                    const monitorStatusBar = MableRow.querySelector("#monitor_status .progress-bar");
                    if (monitorStatusBar) {
                        monitorStatusBar.setAttribute("class", statusClass);
                        monitorStatusBar.innerHTML = `<small>${statusText}</small>`;
                    }
                }

                if (TableRow) {
                    TableRow.querySelector("#name").innerHTML = server.name;
                    TableRow.querySelector("#type").innerHTML = server.type;
                    TableRow.querySelector("#location").innerHTML = server.location;
                }

                if (MableRow) {
                    MableRow.querySelector("#monitor_node").innerHTML = server.name;
                    MableRow.querySelector("#monitor_location").innerHTML = server.location;
                }

                if (!server.online4 && !server.online6) {
                    if (server_status[i]) {
                        if (TableRow) {
                            TableRow.querySelector("#uptime").innerHTML = "–";
                            TableRow.querySelector("#load").innerHTML = "–";
                            TableRow.querySelector("#network").innerHTML = "–";
                            TableRow.querySelector("#traffic").innerHTML = "–";
                            const monthTrafficBar = TableRow.querySelector("#month_traffic .progress-bar");
                            if (monthTrafficBar) {
                                monthTrafficBar.setAttribute("class", "progress-bar bg-warning");
                                monthTrafficBar.innerHTML = "<small>关闭</small>";
                            }
                            const cpuBar = TableRow.querySelector("#cpu .progress-bar");
                            if (cpuBar) {
                                cpuBar.setAttribute("class", "progress-bar bg-danger");
                                cpuBar.style.width = "100%";
                                cpuBar.innerHTML = "<small>关闭</small>";
                            }
                            const memoryBar = TableRow.querySelector("#memory .progress-bar");
                            if (memoryBar) {
                                memoryBar.setAttribute("class", "progress-bar bg-danger");
                                memoryBar.style.width = "100%";
                                memoryBar.innerHTML = "<small>关闭</small>";
                            }
                            const hddBar = TableRow.querySelector("#hdd .progress-bar");
                            if (hddBar) {
                                hddBar.setAttribute("class", "progress-bar bg-danger");
                                hddBar.style.width = "100%";
                                hddBar.innerHTML = "<small>关闭</small>";
                            }
                            const pingBar = TableRow.querySelector("#ping .progress-bar");
                            if (pingBar) {
                                pingBar.setAttribute("class", "progress-bar bg-danger");
                                pingBar.style.width = "100%";
                                pingBar.innerHTML = "<small>关闭</small>";
                            }
                        }
                        if (MableRow) {
                            MableRow.querySelector("#monitor_text").innerHTML = "-";
                        }
                        if (ExpandRow && ExpandRow.classList.contains("show")) ExpandRow.classList.remove("show");
                        if (TableRow) TableRow.setAttribute("data-bs-target", "");
                        if (MableRow) MableRow.setAttribute("data-bs-target", "");
                        server_status[i] = false;
                    }
                } else {
                    if (!server_status[i]) {
                        if (TableRow) TableRow.setAttribute("data-bs-target", `#rt${i}`);
                        if (MableRow) MableRow.setAttribute("data-bs-target", `#rt${i}`);
                        server_status[i] = true;
                    }

                    const trafficdiff_in = server.network_in - server.last_network_in;
                    const trafficdiff_out = server.network_out - server.last_network_out;
                    const monthtraffic = `${bytesToSize(trafficdiff_in, 1, true)} | ${bytesToSize(trafficdiff_out, 1, true)}`;
                    if (TableRow) {
                        const monthTrafficBar = TableRow.querySelector("#month_traffic .progress-bar");
                        if (monthTrafficBar) {
                            monthTrafficBar.setAttribute("class", "progress-bar bg-success");
                            monthTrafficBar.innerHTML = `<small>${monthtraffic}</small>`;
                        }
                    }

                    if (TableRow) TableRow.querySelector("#uptime").innerHTML = server.uptime;
                    if (TableRow) TableRow.querySelector("#load").innerHTML = server.load_1 == -1 ? "–" : server.load_1.toFixed(2);

                    const netstr = `${bytesToSize(server.network_rx, 1, true)} | ${bytesToSize(server.network_tx, 1, true)}`;
                    if (TableRow) TableRow.querySelector("#network").innerHTML = netstr;

                    const trafficstr = `${bytesToSize(server.network_in, 1, true)} | ${bytesToSize(server.network_out, 1, true)}`;
                    if (TableRow) TableRow.querySelector("#traffic").innerHTML = trafficstr;

                    const cpuClass = server.cpu >= 90 ? "progress-bar bg-danger" : server.cpu >= 80 ? "progress-bar bg-warning" : "progress-bar bg-success";
                    if (TableRow) {
                        const cpuBar = TableRow.querySelector("#cpu .progress-bar");
                        if (cpuBar) {
                            cpuBar.setAttribute("class", cpuClass);
                            cpuBar.style.width = `${server.cpu}%`;
                            cpuBar.innerHTML = `${server.cpu}%`;
                        }
                    }

                    const Mem = ((server.memory_used / server.memory_total) * 100).toFixed(0);
                    const memClass = Mem >= 90 ? "progress-bar bg-danger" : Mem >= 80 ? "progress-bar bg-warning" : "progress-bar bg-success";
                    if (TableRow) {
                        const memoryBar = TableRow.querySelector("#memory .progress-bar");
                        if (memoryBar) {
                            memoryBar.setAttribute("class", memClass);
                            memoryBar.style.width = `${Mem}%`;
                            memoryBar.innerHTML = `${Mem}%`;
                        }
                    }
                    if (ExpandRow) ExpandRow.querySelector("#expand_mem").innerHTML = `内存|虚存: ${bytesToSize(server.memory_used * 1024, 1)} / ${bytesToSize(server.memory_total * 1024, 1)} | ${bytesToSize(server.swap_used * 1024, 0)} / ${bytesToSize(server.swap_total * 1024, 0)}`;

                    const HDD = ((server.hdd_used / server.hdd_total) * 100).toFixed(0);
                    const hddClass = HDD >= 90 ? "progress-bar bg-danger" : HDD >= 80 ? "progress-bar bg-warning" : "progress-bar bg-success";
                    if (TableRow) {
                        const hddBar = TableRow.querySelector("#hdd .progress-bar");
                        if (hddBar) {
                            hddBar.setAttribute("class", hddClass);
                            hddBar.style.width = `${HDD}%`;
                            hddBar.innerHTML = `${HDD}%`;
                        }
                    }
                    const io = `${bytesToSize(server.io_read, 0, true)} / ${bytesToSize(server.io_write, 0, true)}`;
                    if (ExpandRow) ExpandRow.querySelector("#expand_hdd").innerHTML = `硬盘|读写: ${bytesToSize(server.hdd_used * 1024 * 1024, 2)} / ${bytesToSize(server.hdd_total * 1024 * 1024, 2)} | ${io}`;

                    if (ExpandRow) ExpandRow.querySelector("#expand_tupd").innerHTML = `TCP/UDP/进/线: ${server.tcp_count} / ${server.udp_count} / ${server.process_count} / ${server.thread_count}`;

                    const PING_10010 = server.ping_10010.toFixed(0);
                    const PING_189 = server.ping_189.toFixed(0);
                    const PING_10086 = server.ping_10086.toFixed(0);
                    const pingClass = PING_10010 >= 20 || PING_189 >= 20 || PING_10086 >= 20 ? "progress-bar bg-danger" : PING_10010 >= 10 || PING_189 >= 10 || PING_10086 >= 10 ? "progress-bar bg-warning" : "progress-bar bg-success";
                    if (TableRow) {
                        const pingBar = TableRow.querySelector("#ping .progress-bar");
                        if (pingBar) {
                            pingBar.setAttribute("class", pingClass);
                            pingBar.innerHTML = `${PING_10010}%💻${PING_189}%💻${PING_10086}%`;
                        }
                    }
                    if (ExpandRow) ExpandRow.querySelector("#expand_ping").innerHTML = `CU/CT/CM: ${server.time_10010}ms (${PING_10010}%) / ${server.time_189}ms (${PING_189}%) / ${server.time_10086}ms (${PING_10086}%)`;

                    if (MableRow) MableRow.querySelector("#monitor_text").innerHTML = server.custom;
                }
            });

            d = new Date(result.updated * 1000);
            error = 0;
        })
        .catch(error => {
            console.error("Fetch error: ", error);
            if (!error) {
                document.querySelectorAll("#servers > tr.accordion-toggle").forEach((TableRow, i) => {
                    const MableRow = document.querySelector(`#monitors tr#r${i}`);
                    const ExpandRow = document.querySelector(`#servers #rt${i}`);

                    if (TableRow && MableRow) {
                        TableRow.querySelectorAll(".progress-bar").forEach(bar => {
                            if (bar) {
                                bar.setAttribute("class", "progress-bar bg-danger");
                                bar.innerHTML = "<small>错误</small>";
                            }
                        });

                        MableRow.querySelectorAll(".progress-bar").forEach(bar => {
                            if (bar) {
                                bar.setAttribute("class", "progress-bar bg-danger");
                                bar.innerHTML = "<small>错误</small>";
                            }
                        });

                        if (ExpandRow && ExpandRow.classList.contains("show")) {
                            ExpandRow.classList.remove("show");
                        }

                        TableRow.setAttribute("data-bs-target", "");
                        MableRow.setAttribute("data-bs-target", "");
                        server_status[i] = false;
                    } else {
                        console.error(`TableRow or MableRow is undefined for index ${i}`);
                    }
                });
            }
            error = 1;
            document.getElementById("updated").innerHTML = "更新错误.";
        });
}

function updateTime() {
    if (!error) document.getElementById("updated").innerHTML = "最后更新: " + timeSince(d);
}

uptime();
updateTime();
// 降低改值，可以减少cpu占用
setInterval(uptime, 2000);
setInterval(updateTime, 2000);

// styleswitcher.js
function setActiveStyleSheet(title) {
    var i, a, main;
    for (i = 0; (a = document.getElementsByTagName("link")[i]); i++) {
        if (a.getAttribute("rel").indexOf("stylesheet") != -1 && a.getAttribute("title")) {
            a.disabled = true;
            if (a.getAttribute("title") == title) a.disabled = false;
        }
    }
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

    // 处理标签页切换
    const tabs = document.querySelectorAll('.nav-link');
    tabs.forEach(tab => {
        tab.addEventListener('click', function(event) {
            if (this.id === 'navbarDropdown') {
                return; // 阻止“风格”标签的默认行为
            }
            event.preventDefault();
            const target = this.getAttribute('href');
            document.querySelectorAll('.tab-pane').forEach(pane => pane.classList.remove('show', 'active'));
            document.querySelector(target).classList.add('show', 'active');
            tabs.forEach(t => t.classList.remove('active'));
            this.classList.add('active');
        });
    });
}