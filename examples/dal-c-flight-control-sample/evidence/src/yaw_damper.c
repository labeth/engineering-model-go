/* ENGMODEL-OWNER-UNIT: FU-YAW-CONTROL */
/* TRLC-LINKS: REQ-YD-HLR-001, REQ-YD-LLR-001 */
/* Example placeholder only; not airborne software. */
double yaw_damper(double yaw_rate) {
    double command = yaw_rate * -0.25;
    if (command > 5.0) return 5.0;
    if (command < -5.0) return -5.0;
    return command;
}
