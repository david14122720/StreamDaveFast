var media, repro, bar, progress, max;
max = 435;

function start() {
	media = document.getElementById("video");
	repro = document.getElementById("repro");
	bar = document.getElementById("bar");
	progress = document.getElementById("progress");

	repro.addEventListener("click", clicking, false);
	bar.addEventListener("click", moving, false);
}

function clicking() {
	if ((media.paused==false) && (media.ended==false)) {
		media.pause();
		repro.innerHTML="Play";
	}
	else {
		media.play();
		repro.innerHTML="Pause";
		bucle = setInterval(status, 50);
	}
}
function status() {
	if (media.ended==false) {
		var total=parseInt(media.currentTime*max/media.duration);
		progress.style.width=total+"px";
	}
}
function moving(position) {
	if ((media.paused==false) && (media.ended)==false) {
		var mousex = position.pageX-bar.offsetLeft;
		var ntime = mousex*media.duration/max;
		media.currentTime = ntime;

		progress.style.width = mousex+"px";
	}
}

window.addEventListener("load", start, false);
