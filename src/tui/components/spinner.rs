const FRAMES: [&str; 8] = ["⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"];

pub struct Spinner {
    frame: usize,
    label: String,
    visible: bool,
}

impl Spinner {
    pub fn new(label: impl Into<String>) -> Self {
        Self {
            frame: 0,
            label: label.into(),
            visible: false,
        }
    }

    pub fn start(&mut self) {
        self.visible = true;
    }

    pub fn stop(&mut self) {
        self.visible = false;
    }

    pub fn set_label(&mut self, label: impl Into<String>) {
        self.label = label.into();
    }

    pub fn tick(&mut self) {
        if self.visible {
            self.frame = (self.frame + 1) % FRAMES.len();
        }
    }

    pub fn view(&self) -> String {
        if !self.visible {
            return String::new();
        }
        format!("{} {}", FRAMES[self.frame], self.label)
    }
}
