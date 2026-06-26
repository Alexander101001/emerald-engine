/// Auto-synthesised adapter from farion1231/cc-switch
struct cc_switchAdapter {
    name: String,
}

impl cc_switchAdapter {
    fn new() -> Self {
        Self { name: "cc_switch".to_string() }
    }

    fn execute(&self) {
        println!("[{}] Executing synthesised capability", self.name);
    }
}
